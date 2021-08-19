package job

import (
	"strings"
	"testing"

	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestNew(t *testing.T) {
	var (
		plan  *upgradeapiv1.Plan
		found bool
	)
	skipValues := []int{-1, 0, 1}
	for _, val := range skipValues {
		t.Run("value", func(t *testing.T) {

			plan = upgradeapiv1.NewPlan("skip-wait", "plan1", upgradeapiv1.Plan{
				Spec: upgradeapiv1.PlanSpec{Drain: &upgradeapiv1.DrainSpec{SkipWaitForDeleteTimeout: val},
					Upgrade: &upgradeapiv1.ContainerSpec{Image: "image"}},
			})
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: labels.Set{
						corev1.LabelHostname: "node1",
					},
				},
			}
			job := New(plan, node, "ctr")
			t.Logf("%#v", job.Spec.Template.Spec.InitContainers)
			for _, container := range job.Spec.Template.Spec.InitContainers {
				if container.Name == "drain" {
					for _, arg := range container.Args {
						if strings.Contains(arg, "--skip-wait-for-delete-timeout") {
							found = true
							break
						}
					}
					switch val {
					case 1:
						if !found {
							t.Errorf("tag doesnt exist")
						}
					default:
						if found {
							t.Errorf("tag shouldnt exist")
						}
					}
				}
			}

		})
	}
}
