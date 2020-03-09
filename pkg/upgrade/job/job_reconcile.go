package job

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/rancher/wrangler/pkg/apply"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Reconciler returns values suitable for use in apply.WithReconciler
func Reconciler() (schema.GroupVersionKind, apply.Reconciler) {
	convert := func(src interface{}, obj interface{}) error {
		u, ok := src.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("expected unstructured but got %v", reflect.TypeOf(src))
		}
		bytes, err := u.MarshalJSON()
		if err != nil {
			return err
		}
		return json.Unmarshal(bytes, obj)
	}
	// adapted from wrangler's apply.reconcileJob
	return batchv1.SchemeGroupVersion.WithKind("Job"), func(oldObj runtime.Object, newObj runtime.Object) (b bool, err error) {
		oldJob, ok := oldObj.(*batchv1.Job)
		if !ok {
			oldJob = &batchv1.Job{}
			if err := convert(oldObj, oldJob); err != nil {
				return false, err
			}
		}

		if ConditionFailed.IsTrue(oldJob) {
			return false, apply.ErrReplace
		}

		newJob, ok := newObj.(*batchv1.Job)
		if !ok {
			newJob = &batchv1.Job{}
			if err := convert(newObj, newJob); err != nil {
				return false, err
			}
		}

		if !equality.Semantic.DeepEqual(oldJob.Spec.Template, newJob.Spec.Template) {
			return false, apply.ErrReplace
		}

		return false, nil
	}
}
