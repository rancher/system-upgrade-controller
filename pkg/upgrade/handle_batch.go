package upgrade

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradejob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	batchctlv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// job events (successful completions) cause the node the job ran on to be labeled as per the plan
func (ctl *Controller) handleJobs(ctx context.Context) error {
	plans := ctl.upgradeFactory.Upgrade().V1().Plan()
	nodes := ctl.coreFactory.Core().V1().Node()
	jobs := ctl.batchFactory.Batch().V1().Job()

	jobs.OnChange(ctx, ctl.Name, func(_ string, obj *batchv1.Job) (*batchv1.Job, error) {
		if obj == nil {
			return obj, nil
		}
		jobSelector := labels.SelectorFromSet(labels.Set{
			upgradeapi.LabelController: ctl.Name,
		})
		// avoid commandeering jobs from other controllers
		if obj.Labels == nil || !jobSelector.Matches(labels.Set(obj.Labels)) {
			return obj, nil
		}
		// identify the plan that this job is applying
		planName, ok := obj.Labels[upgradeapi.LabelPlan]
		if !ok {
			// malformed, just delete it and move on
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		}
		// what version of the plan is this job applying?
		planVersion, ok := obj.Labels[upgradeapi.LabelVersion]
		if !ok {
			// malformed, just delete it and move on
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		}
		// get the plan being applied
		plan, err := plans.Get(obj.Namespace, planName, metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			// plan is gone, delete
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		case err != nil:
			return obj, err
		}
		// if this job was applying a different version then just delete it
		// this has the side-effect of only ever retaining one job per node during the TTL window
		if planVersion != plan.Status.LatestVersion {
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		}
		// trigger the plan when we're done, might free up a concurrency slot
		logrus.Debugf("Enqueing sync of Plan %s/%s from Job %s/%s", obj.Namespace, planName, obj.Namespace, obj.Name)
		defer plans.Enqueue(obj.Namespace, planName)
		// identify the node that this job is targeting
		nodeName, ok := obj.Labels[upgradeapi.LabelNode]
		if !ok {
			// malformed, just delete it and move on
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		}
		// get the node that the plan is being applied to
		node, err := nodes.Cache().Get(nodeName)
		switch {
		case apierrors.IsNotFound(err):
			// node is gone, delete
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		case err != nil:
			return obj, err
		}
		// if the job has failed enqueue-or-delete it depending on the TTL window
		if upgradejob.ConditionFailed.IsTrue(obj) {
			failedTime := upgradejob.ConditionFailed.GetLastTransitionTime(obj)
			if failedTime.IsZero() {
				return obj, fmt.Errorf("condition %q missing field %q", upgradejob.ConditionFailed, "LastTransitionTime")
			}
			message := fmt.Sprintf("Job %s/%s failed on Node %s: %s: %s",
				obj.Namespace, obj.Name, nodeName,
				upgradejob.ConditionFailed.GetReason(obj),
				upgradejob.ConditionFailed.GetMessage(obj),
			)
			ctl.recorder.Eventf(plan, corev1.EventTypeWarning, "JobFailed", message)
			upgradeapiv1.PlanComplete.SetError(plan, "JobFailed", errors.New(message))
			if plan, err = plans.UpdateStatus(plan); err != nil {
				return obj, err
			}
			return obj, enqueueOrDelete(jobs, obj, failedTime)
		}
		// if the job has completed tag the node then enqueue-or-delete depending on the TTL window
		if upgradejob.ConditionComplete.IsTrue(obj) {
			completeTime := upgradejob.ConditionComplete.GetLastTransitionTime(obj)
			if completeTime.IsZero() {
				return obj, fmt.Errorf("condition %q missing field %q", upgradejob.ConditionComplete, "LastTransitionTime")
			}
			planLabel := upgradeapi.LabelPlanName(planName)
			if planHash, ok := obj.Labels[planLabel]; ok {
				var delay time.Duration
				if plan.Spec.PostCompleteDelay != nil {
					delay = plan.Spec.PostCompleteDelay.Duration
				}
				// if the job has not been completed for the configured delay, re-enqueue
				// it for processing once the delay has elapsed.
				// the job's TTLSecondsAfterFinished is guaranteed to be set to a larger value
				// than the plan's requested delay.
				if interval := time.Now().Sub(completeTime); interval < delay {
					logrus.Debugf("Enqueing sync of Job %s/%s in %v", obj.Namespace, obj.Name, delay-interval)
					ctl.recorder.Eventf(plan, corev1.EventTypeNormal, "JobCompleteWaiting", "Job completed on Node %s, waiting %s PostCompleteDelay", node.Name, delay)
					jobs.EnqueueAfter(obj.Namespace, obj.Name, delay-interval)
				} else {
					ctl.recorder.Eventf(plan, corev1.EventTypeNormal, "JobComplete", "Job completed on Node %s", node.Name)
					node.Labels[planLabel] = planHash
				}
				// mark the node as schedulable even if the delay has not elapsed, so that
				// workloads can resume scheduling.
				if node.Spec.Unschedulable && (plan.Spec.Cordon || plan.Spec.Drain != nil) {
					node.Spec.Unschedulable = false
				}
				if node, err = nodes.Update(node); err != nil {
					return obj, err
				}
			}
			return obj, enqueueOrDelete(jobs, obj, completeTime)
		}
		// if the job is hasn't failed or completed but the job Node is not on the applying list, consider it running out-of-turn and delete it
		if i := sort.SearchStrings(plan.Status.Applying, nodeName); i == len(plan.Status.Applying) ||
			(i < len(plan.Status.Applying) && plan.Status.Applying[i] != nodeName) {
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		}
		return obj, nil
	})

	return nil
}

func enqueueOrDelete(jobController batchctlv1.JobController, job *batchv1.Job, lastTransitionTime time.Time) error {
	var ttlSecondsAfterFinished time.Duration

	if job.Spec.TTLSecondsAfterFinished == nil {
		if annotation, ok := job.Annotations[upgradeapi.AnnotationTTLSecondsAfterFinished]; ok {
			fallbackTTLSecondsAfterFinished, err := strconv.ParseInt(annotation, 10, 32)
			if err != nil {
				// malformed, delete
				return deleteJob(jobController, job, metav1.DeletePropagationBackground)
			}
			ttlSecondsAfterFinished = time.Second * time.Duration(fallbackTTLSecondsAfterFinished)
		}
	} else {
		ttlSecondsAfterFinished = time.Second * time.Duration(*job.Spec.TTLSecondsAfterFinished)
	}
	if interval := time.Now().Sub(lastTransitionTime); interval < ttlSecondsAfterFinished {
		logrus.Debugf("Enqueing sync of Job %s/%s in %v", job.Namespace, job.Name, ttlSecondsAfterFinished-interval)
		jobController.EnqueueAfter(job.Namespace, job.Name, ttlSecondsAfterFinished-interval)
		return nil
	}
	return deleteJob(jobController, job, metav1.DeletePropagationBackground)
}

func deleteJob(jobController batchctlv1.JobController, job *batchv1.Job, deletionPropagation metav1.DeletionPropagation) error {
	return jobController.Delete(job.Namespace, job.Name, &metav1.DeleteOptions{PropagationPolicy: &deletionPropagation})
}
