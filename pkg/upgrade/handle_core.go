package upgrade

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// node events with labels that match a plan's selectors (potentially) trigger that plan
func (ctl *Controller) handleNodes(ctx context.Context) error {
	plans := ctl.upgradeFactory.Upgrade().V1().Plan()

	ctl.coreFactory.Core().V1().Node().OnChange(ctx, ctl.Name, func(_ string, obj *corev1.Node) (*corev1.Node, error) {
		if obj == nil {
			return obj, nil
		}
		planList, err := plans.Cache().List(ctl.Namespace, labels.Everything())
		if err != nil {
			return obj, err
		}
		for _, plan := range planList {
			if selector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector); err != nil {
				return obj, err
			} else if selector.Matches(labels.Set(obj.Labels)) {
				plans.Enqueue(plan.Namespace, plan.Name)
			}
		}
		return obj, nil
	})

	return nil
}

// secret events referred to by a plan (potentially) trigger that plan
func (ctl *Controller) handleSecrets(ctx context.Context) error {
	plans := ctl.upgradeFactory.Upgrade().V1().Plan()

	ctl.coreFactory.Core().V1().Secret().OnChange(ctx, ctl.Name, func(_ string, obj *corev1.Secret) (*corev1.Secret, error) {
		if obj == nil {
			return obj, nil
		}
		planList, err := plans.Cache().List(ctl.Namespace, labels.Everything())
		if err != nil {
			return obj, err
		}
		for _, plan := range planList {
			for _, secret := range plan.Spec.Secrets {
				if obj.Name == secret.Name {
					if !secret.IgnoreUpdates {
						plans.Enqueue(plan.Namespace, plan.Name)
						continue
					}
				}
			}
		}
		return obj, nil
	})

	return nil
}
