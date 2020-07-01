package job

import (
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"strings"
	"testing"
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
			job := New(plan, "node1", "ctr")
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
