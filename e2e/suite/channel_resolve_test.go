package suite_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradeplan "github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
)

var _ = Describe("Resolve channel", func() {
	e2e := framework.New("channel")

	When("passed url fails to resolve", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			ctx        context.Context
			channelSrv *httptest.Server
			clusterID  string
			latest     string
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("channel-", "", nil)
		})
		AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
		})
		It("channel server is up with correct address", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusFound)
			plan.Spec.Channel = channelSrv.URL
			Expect(plan.Spec.Channel).ToNot(BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			Expect(latest).NotTo(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})
		It("channel server is up but url not found", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusNotFound)
			plan.Spec.Channel = channelSrv.URL
			Expect(plan.Spec.Channel).ToNot(BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			Expect(latest).To(BeEmpty())
			Expect(err).To(HaveOccurred())
		})
		It("Service Unavailable", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusServiceUnavailable)
			plan.Spec.Channel = channelSrv.URL
			Expect(plan.Spec.Channel).ToNot(BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			Expect(latest).To(BeEmpty())
			Expect(err).To(HaveOccurred())
		})
	})
})
