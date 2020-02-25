package suite_test

import (
	"net/http/httptest"
	"path"
	"time"

	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
)

var _ = Describe("Plan Resolution", func() {
	e2e := framework.New("resolve")

	When("missing channel and version", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("missing-", "", nil)
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.MatchesError(plan, "Error", upgradeapiv1.ErrPlanUnresolvable)).To(BeTrue())
			Expect(plan.Status.LatestVersion).To(BeEmpty())
			Expect(plan.Status.LatestHash).To(BeEmpty())
		})
	})

	When("has version", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("version-", "", nil)
			plan.Spec.Version = "test"

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Version"))
			Expect(plan.Status.LatestVersion).To(Equal(plan.Spec.Version))
			Expect(plan.Status.LatestHash).ToNot(BeEmpty())
		})
	})

	When("has version with semver+metadata", func() {
		var (
			err    error
			plan   *upgradeapiv1.Plan
			semver = "v1.2.3+test"
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("version-semver-metadata-", "", nil)
			plan.Spec.Version = semver

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Version"))
			Expect(plan.Status.LatestHash).ToNot(BeEmpty())
		})
		It("should munge the semver", func() {
			Expect(plan.Status.LatestVersion).ToNot(ContainSubstring(`+`))
		})
	})

	When("has channel", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			channelSrv *httptest.Server
			channelTag = "test"
		)
		BeforeEach(func() {
			channelSrv = framework.ChannelServer(path.Join("/local", channelTag))
			plan = e2e.NewPlan("channel-", "", nil)
			plan.Spec.Channel = channelSrv.URL
			Expect(plan.Spec.Channel).ToNot(BeEmpty())

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Channel"))
			Expect(plan.Status.LatestVersion).To(Equal(channelTag))
			Expect(plan.Status.LatestHash).ToNot(BeEmpty())
		})
	})

	When("has channel with semver+metadata", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			channelSrv *httptest.Server
			channelTag = "v1.2.3+test"
		)
		BeforeEach(func() {
			channelSrv = framework.ChannelServer(path.Join("/local/test", channelTag))
			plan = e2e.NewPlan("channel-semver-metadata-", "", nil)
			plan.Spec.Channel = channelSrv.URL
			Expect(plan.Spec.Channel).ToNot(BeEmpty())

			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
		})
		It("should resolve", func() {
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())
			Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(Equal("Channel"))
			Expect(plan.Status.LatestHash).ToNot(BeEmpty())
		})
		It("should munge the semver", func() {
			Expect(plan.Status.LatestVersion).ToNot(ContainSubstring(`+`))
		})
	})
})
