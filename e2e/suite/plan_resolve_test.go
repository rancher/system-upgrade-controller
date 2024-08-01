package suite_test

import (
	"net/http"
	"net/http/httptest"
	"path"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
)

var _ = ginkgo.Describe("Plan Resolution", func() {
	e2e := framework.New("resolve")

	ginkgo.When("missing channel and version", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
		)
		ginkgo.BeforeEach(func() {
			plan = e2e.NewPlan("missing-", "", nil)
			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.It("should not resolve", func() {
			gomega.Expect(upgradeapiv1.PlanLatestResolved.MatchesError(plan, "Error", upgradeapiv1.ErrPlanUnresolvable)).To(gomega.BeTrue())
			gomega.Expect(plan.Status.LatestVersion).To(gomega.BeEmpty())
			gomega.Expect(plan.Status.LatestHash).To(gomega.BeEmpty())
		})
	})

	ginkgo.When("has version", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
		)
		ginkgo.BeforeEach(func() {
			plan = e2e.NewPlan("version-", "", nil)
			plan.Spec.Version = "test"

			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.It("should resolve", func() {
			gomega.Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(gomega.BeTrue())
			gomega.Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(gomega.Equal("Version"))
			gomega.Expect(plan.Status.LatestVersion).To(gomega.Equal(plan.Spec.Version))
			gomega.Expect(plan.Status.LatestHash).ToNot(gomega.BeEmpty())
		})
	})

	ginkgo.When("has version with semver+metadata", func() {
		var (
			err    error
			plan   *upgradeapiv1.Plan
			semver = "v1.2.3+test"
		)
		ginkgo.BeforeEach(func() {
			plan = e2e.NewPlan("version-semver-metadata-", "", nil)
			plan.Spec.Version = semver

			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.It("should resolve", func() {
			gomega.Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(gomega.BeTrue())
			gomega.Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(gomega.Equal("Version"))
			gomega.Expect(plan.Status.LatestHash).ToNot(gomega.BeEmpty())
		})
		ginkgo.It("should munge the semver", func() {
			gomega.Expect(plan.Status.LatestVersion).ToNot(gomega.ContainSubstring(`+`))
		})
	})

	ginkgo.When("has channel", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			channelSrv *httptest.Server
			channelTag = "test"
		)
		ginkgo.BeforeEach(func() {
			channelSrv = framework.ChannelServer(path.Join("/local", channelTag), http.StatusFound)
			plan = e2e.NewPlan("channel-", "", nil)
			plan.Spec.Channel = channelSrv.URL
			gomega.Expect(plan.Spec.Channel).ToNot(gomega.BeEmpty())

			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
		})
		ginkgo.It("should resolve", func() {
			gomega.Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(gomega.BeTrue())
			gomega.Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(gomega.Equal("Channel"))
			gomega.Expect(plan.Status.LatestVersion).To(gomega.Equal(channelTag))
			gomega.Expect(plan.Status.LatestHash).ToNot(gomega.BeEmpty())
		})
	})

	ginkgo.When("has channel with semver+metadata", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			channelSrv *httptest.Server
			channelTag = "v1.2.3+test"
		)
		ginkgo.BeforeEach(func() {
			channelSrv = framework.ChannelServer(path.Join("/local/test", channelTag), http.StatusFound)
			plan = e2e.NewPlan("channel-semver-metadata-", "", nil)
			plan.Spec.Channel = channelSrv.URL
			gomega.Expect(plan.Spec.Channel).ToNot(gomega.BeEmpty())

			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
		})
		ginkgo.It("should resolve", func() {
			gomega.Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(gomega.BeTrue())
			gomega.Expect(upgradeapiv1.PlanLatestResolved.GetReason(plan)).To(gomega.Equal("Channel"))
			gomega.Expect(plan.Status.LatestHash).ToNot(gomega.BeEmpty())
		})
		ginkgo.It("should munge the semver", func() {
			gomega.Expect(plan.Status.LatestVersion).ToNot(gomega.ContainSubstring(`+`))
		})
	})
})
