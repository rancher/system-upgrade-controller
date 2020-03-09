package upgrade

import (
	"context"
	"fmt"
	"time"

	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	"github.com/rancher/system-upgrade-controller/pkg/condition"
	upgradejob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	batchctlv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// job events (successful completions) cause the node the job ran on to be labeled as per the plan
func (ctl *Controller) handleJobs(ctx context.Context) error {
	plans := ctl.upgradeFactory.Upgrade().V1().Plan()
	nodes := ctl.coreFactory.Core().V1().Node()
	jobs := ctl.batchFactory.Batch().V1().Job()

	jobs.OnChange(ctx, ctl.Name, func(key string, obj *batchv1.Job) (*batchv1.Job, error) {
		if obj == nil {
			return obj, nil
		}
		jobSelector := labels.SelectorFromSet(labels.Set{
			upgradeapi.LabelController: ctl.Name,
		})
		// avoid commandeering jobs that don't belong to us
		if obj.Labels != nil && jobSelector.Matches(labels.Set(obj.Labels)) {
			if planName, ok := obj.Labels[upgradeapi.LabelPlan]; ok {
				defer plans.Enqueue(obj.Namespace, planName)
				if upgradejob.ConditionComplete.IsTrue(obj) {
					planLabel := upgradeapi.LabelPlanName(planName)
					if planHash, ok := obj.Labels[planLabel]; ok {
						if nodeName, ok := obj.Labels[upgradeapi.LabelNode]; ok {
							node, err := nodes.Cache().Get(nodeName)
							if err != nil {
								return obj, err
							}
							plan, err := plans.Cache().Get(obj.Namespace, planName)
							if err != nil {
								return obj, err
							}
							node.Labels[planLabel] = planHash
							if node.Spec.Unschedulable && (plan.Spec.Cordon || plan.Spec.Drain != nil) {
								node.Spec.Unschedulable = false
							}
							if node, err = nodes.Update(node); err != nil {
								return obj, err
							}
						}
					}
				}
				if upgradejob.ConditionComplete.IsTrue(obj) {
					return obj, enqueueOrDelete(jobs, obj, upgradejob.ConditionComplete)
				}
				if upgradejob.ConditionFailed.IsTrue(obj) {
					return obj, enqueueOrDelete(jobs, obj, upgradejob.ConditionFailed)
				}
			}
		}
		return obj, nil
	})

	return nil
}

func enqueueOrDelete(jobController batchctlv1.JobController, job *batchv1.Job, done condition.Cond) error {
	if job.Spec.TTLSecondsAfterFinished == nil {
		return nil
	}
	lastTransitionTime := done.GetLastTransitionTime(job)
	if lastTransitionTime.IsZero() {
		return fmt.Errorf("condition %q missing field %q", done, "LastTransitionTime")
	}
	ttlSecondsAfterFinished := time.Second * time.Duration(*job.Spec.TTLSecondsAfterFinished)
	if interval := time.Now().Sub(lastTransitionTime); interval < ttlSecondsAfterFinished {
		jobController.EnqueueAfter(job.Namespace, job.Name, ttlSecondsAfterFinished-interval)
		return nil
	}
	deletePropagationBackground := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{PropagationPolicy: &deletePropagationBackground}
	return jobController.Delete(job.Namespace, job.Name, &deleteOptions)
}
