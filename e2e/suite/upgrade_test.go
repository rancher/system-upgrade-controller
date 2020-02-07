package suite_test

import (
	"time"

	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
)

var _ = Describe("Upgrade", func() {
	e2e := framework.New("upgrade")
	When("plan missing channel and version", func() {
		var (
			err  error
			plan = e2e.NewPlan("unresolvable-", "", nil)
		)
		BeforeEach(func() {
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.PollPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 2*time.Second, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.MatchesError(plan, "Error", upgradeapiv1.ErrPlanUnresolvable)).To(BeTrue())
		})
	})
	When("plan has version", func() {
		var (
			err  error
			plan = e2e.NewPlan("resolve-version-", "", nil)
		)
		BeforeEach(func() {
			plan.Spec.Version = "test"

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.PollPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 2*time.Second, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Version"))
		})
	})
	When("plan has channel", func() {
		var (
			err  error
			plan = e2e.NewPlan("resolve-channel-", "", nil)
		)
		BeforeEach(func() {
			plan.Spec.Channel = e2e.ChannelServerURL()
			Expect(plan.Spec.Channel).ToNot(BeEmpty())

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.PollPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 2*time.Second, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Channel"))
		})
	})
})
