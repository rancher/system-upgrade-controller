package suite_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradeplan "github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
)

var _ = ginkgo.Describe("Resolve channel", func() {
	e2e := framework.New("channel")

	ginkgo.When("passed url fails to resolve", func() {
		var (
			err        error
			plan       *upgradeapiv1.Plan
			ctx        context.Context
			cancel     context.CancelFunc
			channelSrv *httptest.Server
			clusterID  string
			latest     string
		)
		ginkgo.BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			plan = e2e.NewPlan("channel-", "", nil)
		})
		ginkgo.AfterEach(func() {
			if channelSrv != nil {
				channelSrv.Close()
			}
			cancel()
		})
		ginkgo.It("channel server is up with correct address", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusFound)
			plan.Spec.Channel = channelSrv.URL
			gomega.Expect(plan.Spec.Channel).ToNot(gomega.BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(latest).NotTo(gomega.BeEmpty())
		})
		ginkgo.It("channel server is up but url not found", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusNotFound)
			plan.Spec.Channel = channelSrv.URL
			gomega.Expect(plan.Spec.Channel).ToNot(gomega.BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(latest).To(gomega.BeEmpty())
		})
		ginkgo.It("Service Unavailable", func() {
			channelSrv = framework.ChannelServer("/local", http.StatusServiceUnavailable)
			plan.Spec.Channel = channelSrv.URL
			gomega.Expect(plan.Spec.Channel).ToNot(gomega.BeEmpty())
			plan, err = e2e.CreatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			latest, err = upgradeplan.ResolveChannel(ctx, plan.Spec.Channel, plan.Status.LatestVersion, clusterID)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(latest).To(gomega.BeEmpty())
		})
	})
})
