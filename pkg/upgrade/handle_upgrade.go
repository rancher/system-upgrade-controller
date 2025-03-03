package upgrade

import (
	"context"
	"slices"
	"strings"
	"time"

	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradectlv1 "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io/v1"
	upgradejob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	upgradenode "github.com/rancher/system-upgrade-controller/pkg/upgrade/node"
	upgradeplan "github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (ctl *Controller) handlePlans(ctx context.Context) error {
	jobs := ctl.batchFactory.Batch().V1().Job()
	nodes := ctl.coreFactory.Core().V1().Node()
	plans := ctl.upgradeFactory.Upgrade().V1().Plan()
	secrets := ctl.coreFactory.Core().V1().Secret()
	secretsCache := secrets.Cache()
	recorder := ctl.recorder

	// process plan events, mutating status accordingly
	upgradectlv1.RegisterPlanStatusHandler(ctx, plans, "", ctl.Name,
		func(obj *upgradeapiv1.Plan, status upgradeapiv1.PlanStatus) (upgradeapiv1.PlanStatus, error) {
			if obj == nil {
				return status, nil
			}
			logrus.Debugf("PLAN STATUS HANDLER: plan=%s/%s@%s, status=%+v", obj.Namespace, obj.Name, obj.ResourceVersion, status)

			// ensure that the complete status is present
			complete := upgradeapiv1.PlanComplete
			complete.CreateUnknownIfNotExists(obj)

			// validate plan, and generate events for transitions
			validated := upgradeapiv1.PlanSpecValidated
			validated.CreateUnknownIfNotExists(obj)
			if err := upgradeplan.Validate(obj, secretsCache); err != nil {
				if !validated.IsFalse(obj) {
					recorder.Eventf(obj, corev1.EventTypeWarning, "ValidateFailed", "Failed to validate plan: %v", err)
				}
				validated.SetError(obj, "Error", err)
				return upgradeplan.DigestStatus(obj, secretsCache)
			}
			if !validated.IsTrue(obj) {
				recorder.Event(obj, corev1.EventTypeNormal, "Validated", "Plan is valid")
			}
			validated.SetError(obj, "PlanIsValid", nil)

			// resolve version from spec or channel, and generate events for transitions
			resolved := upgradeapiv1.PlanLatestResolved
			resolved.CreateUnknownIfNotExists(obj)
			// raise error if neither version nor channel are set. this is handled separate from other validation.
			if obj.Spec.Version == "" && obj.Spec.Channel == "" {
				if !resolved.IsFalse(obj) {
					recorder.Event(obj, corev1.EventTypeWarning, "ResolveFailed", upgradeapiv1.ErrPlanUnresolvable.Error())
				}
				resolved.SetError(obj, "Error", upgradeapiv1.ErrPlanUnresolvable)
				return upgradeplan.DigestStatus(obj, secretsCache)
			}
			// use static version from spec if set
			if obj.Spec.Version != "" {
				latest := upgradeplan.MungeVersion(obj.Spec.Version)
				if !resolved.IsTrue(obj) || obj.Status.LatestVersion != latest {
					// Version has changed, set complete to false and emit event
					recorder.Eventf(obj, corev1.EventTypeNormal, "Resolved", "Resolved latest version from Spec.Version: %s", latest)
					complete.False(obj)
					complete.Message(obj, "")
					complete.Reason(obj, "Resolved")
				}
				obj.Status.LatestVersion = latest
				resolved.SetError(obj, "Version", nil)
				return upgradeplan.DigestStatus(obj, secretsCache)
			}
			// re-enqueue a sync at the next channel polling interval, if the LastUpdated time
			// on the resolved status indicates that the interval has not been reached.
			if resolved.IsTrue(obj) {
				if lastUpdated, err := time.Parse(time.RFC3339, resolved.GetLastUpdated(obj)); err == nil {
					if interval := time.Now().Sub(lastUpdated); interval < upgradeplan.PollingInterval {
						plans.EnqueueAfter(obj.Namespace, obj.Name, upgradeplan.PollingInterval-interval)
						return status, nil
					}
				}
			}
			// no static version, poll the channel to get latest version
			latest, err := upgradeplan.ResolveChannel(ctx, obj.Spec.Channel, obj.Status.LatestVersion, ctl.clusterID)
			if err != nil {
				if !resolved.IsFalse(obj) {
					recorder.Eventf(obj, corev1.EventTypeWarning, "ResolveFailed", "Failed to resolve latest version from Spec.Channel: %v", err)
				}
				return status, err
			}
			latest = upgradeplan.MungeVersion(latest)
			if !resolved.IsTrue(obj) || obj.Status.LatestVersion != latest {
				// Version has changed, set complete to false and emit event
				recorder.Eventf(obj, corev1.EventTypeNormal, "Resolved", "Resolved latest version from Spec.Channel: %s", latest)
				complete.False(obj)
				complete.Message(obj, "")
				complete.Reason(obj, "Resolved")
			}
			obj.Status.LatestVersion = latest
			resolved.SetError(obj, "Channel", nil)
			return upgradeplan.DigestStatus(obj, secretsCache)
		},
	)

	// process plan events by creating jobs to apply the plan
	upgradectlv1.RegisterPlanGeneratingHandler(ctx, plans, ctl.apply.WithCacheTypes(nodes, secrets).WithGVK(jobs.GroupVersionKind()).WithDynamicLookup().WithNoDelete(), "", ctl.Name,
		func(obj *upgradeapiv1.Plan, status upgradeapiv1.PlanStatus) (objects []runtime.Object, _ upgradeapiv1.PlanStatus, _ error) {
			if obj == nil {
				return objects, status, nil
			}

			logrus.Debugf("PLAN GENERATING HANDLER: plan=%s/%s@%s, status=%+v", obj.Namespace, obj.Name, obj.ResourceVersion, status)
			// return early without selecting nodes if the plan is not validated and resolved
			complete := upgradeapiv1.PlanComplete
			if !upgradeapiv1.PlanSpecValidated.IsTrue(obj) || !upgradeapiv1.PlanLatestResolved.IsTrue(obj) {
				complete.SetError(obj, "NotReady", ErrPlanNotReady)
				return objects, status, nil
			}

			// select nodes to apply the plan on based on nodeSelector, plan hash, and concurrency
			concurrentNodes, err := upgradeplan.SelectConcurrentNodes(obj, nodes.Cache())
			if err != nil {
				recorder.Eventf(obj, corev1.EventTypeWarning, "SelectNodesFailed", "Failed to select Nodes: %v", err)
				complete.SetError(obj, "SelectNodesFailed", err)
				return objects, status, err
			}

			// Create an upgrade job for each node, and add the node name to Status.Applying
			// Note that this initially creates paused jobs, and then on a second pass once
			// the node has been added to Status.Applying the job parallelism is patched to 1
			// to unpause the job. Ref: https://github.com/rancher/system-upgrade-controller/issues/134
			concurrentNodeNames := make([]string, len(concurrentNodes))
			for i := range concurrentNodes {
				node := concurrentNodes[i]
				objects = append(objects, upgradejob.New(obj, node, ctl.Name))
				concurrentNodeNames[i] = upgradenode.Hostname(node)
			}

			if len(concurrentNodeNames) > 0 {
				// Don't start creating Jobs for the Plan if we're outside the window; just
				// enqueue the plan to check again in a minute to see if we're within the window yet.
				// The Plan is allowed to continue processing as long as there are nodes in progress.
				if window := obj.Spec.Window; window != nil {
					if len(obj.Status.Applying) == 0 && !window.Contains(time.Now()) {
						if complete.GetReason(obj) != "Waiting" {
							recorder.Eventf(obj, corev1.EventTypeNormal, "Waiting", "Waiting for start of Spec.Window to sync Jobs for version %s. Hash: %s", obj.Status.LatestVersion, obj.Status.LatestHash)
						}
						plans.EnqueueAfter(obj.Namespace, obj.Name, time.Minute)
						complete.SetError(obj, "Waiting", ErrOutsideWindow)
						return nil, status, nil
					}
				}

				// If the node list has changed, update Applying status with new node list and emit an event
				if !slices.Equal(obj.Status.Applying, concurrentNodeNames) {
					recorder.Eventf(obj, corev1.EventTypeNormal, "SyncJob", "Jobs synced for version %s on Nodes %s. Hash: %s",
						obj.Status.LatestVersion, strings.Join(concurrentNodeNames, ","), obj.Status.LatestHash)
				}
				obj.Status.Applying = concurrentNodeNames[:]
				complete.False(obj)
				complete.Message(obj, "")
				complete.Reason(obj, "SyncJob")
			} else {
				// set PlanComplete to true when no nodes have been selected,
				// and emit an event if the plan just completed
				if !complete.IsTrue(obj) {
					recorder.Eventf(obj, corev1.EventTypeNormal, "Complete", "Jobs complete for version %s. Hash: %s",
						obj.Status.LatestVersion, obj.Status.LatestHash)
				}
				obj.Status.Applying = nil
				complete.SetError(obj, "Complete", nil)
			}

			return objects, obj.Status, nil
		},
		&generic.GeneratingHandlerOptions{
			AllowClusterScoped:            true,
			NoOwnerReference:              true,
			UniqueApplyForResourceVersion: true,
		},
	)

	return nil
}
