package upgrade

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/rancher/system-upgrade-controller/pkg/apis/condition"
	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradejob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	batchctlv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
		case errors.IsNotFound(err):
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
		case errors.IsNotFound(err):
			// node is gone, delete
			return obj, deleteJob(jobs, obj, metav1.DeletePropagationBackground)
		case err != nil:
			return obj, err
		}
		// if the job has failed enqueue-or-delete it depending on the TTL window
		if upgradejob.ConditionFailed.IsTrue(obj) {
			return obj, enqueueOrDelete(jobs, obj, upgradejob.ConditionFailed)
		}
		// if the job has completed tag the node then enqueue-or-delete depending on the TTL window
		if upgradejob.ConditionComplete.IsTrue(obj) {
			planLabel := upgradeapi.LabelPlanName(planName)
			if planHash, ok := obj.Labels[planLabel]; ok {
				node.Labels[planLabel] = planHash
				if node.Spec.Unschedulable && (plan.Spec.Cordon || plan.Spec.Drain != nil) {
					node.Spec.Unschedulable = false
				}
				if node, err = nodes.Update(node); err != nil {
					return obj, err
				}
			}
			return obj, enqueueOrDelete(jobs, obj, upgradejob.ConditionComplete)
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

func enqueueOrDelete(jobController batchctlv1.JobController, job *batchv1.Job, done condition.Cond) error {
	lastTransitionTime := done.GetLastTransitionTime(job)
	if lastTransitionTime.IsZero() {
		return fmt.Errorf("condition %q missing field %q", done, "LastTransitionTime")
	}

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
