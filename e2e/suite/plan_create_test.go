package suite_test

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
)

var _ = ginkgo.Describe("Plan Creation", func() {
	e2e := framework.New("create")
	ginkgo.When("missing upgrade field", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
		)
		ginkgo.BeforeEach(func() {
			plan = e2e.NewPlan("upgrade", "", nil)
			plan.Spec.Upgrade = nil
			plan, err = e2e.CreatePlan(plan)
		})
		ginkgo.It("should return an error if upgrade in nil", func() {
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})
})
