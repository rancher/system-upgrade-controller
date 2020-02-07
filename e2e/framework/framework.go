package framework

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/rancher/system-upgrade-controller/e2e/framework/controller"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgrade "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned"
	upgradescheme "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/scheme"
	"github.com/rancher/wrangler/pkg/condition"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"
	"k8s.io/kubernetes/test/e2e/framework"
	frameworkauth "k8s.io/kubernetes/test/e2e/framework/auth"
)

type Option func(*Options)

type Options struct {
	framework.Options
}

type Client struct {
	framework.Framework

	UpgradeClientSet *upgrade.Clientset

	controllerDeployment     *appsv1.Deployment
	controllerServiceAccount *corev1.ServiceAccount

	channelServer *httptest.Server
}

func New(name string, opt ...Option) *Client {
	options := &Options{
		Options: framework.Options{
			ClientQPS:   20,
			ClientBurst: 50,
		},
	}
	for _, fn := range opt {
		fn(options)
	}
	client := &Client{
		Framework: framework.Framework{
			BaseName:                 name,
			AddonResourceConstraints: make(map[string]framework.ResourceConstraint),
			Options:                  options.Options,
		},
	}
	ginkgo.BeforeEach(client.BeforeEach)
	ginkgo.AfterEach(client.AfterEach)
	return client
}

func (c *Client) NewPlan(name, image string, command []string, args ...string) *upgradeapiv1.Plan {
	plan := upgradeapiv1.NewPlan("", "", upgradeapiv1.Plan{
		Spec: upgradeapiv1.PlanSpec{
			Upgrade: &upgradeapiv1.ContainerSpec{
				Image:   image,
				Command: command,
				Args:    args,
			},
		},
	})
	if strings.HasSuffix(name, `-`) {
		plan.GenerateName = name
	} else {
		plan.Name = name
	}
	return plan
}

func (c *Client) ChannelServerURL() string {
	return c.channelServer.URL
}

func (c *Client) CreatePlan(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Create(plan)
}

func (c *Client) UpdatePlan(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Update(plan)
}

func (c *Client) UpdatePlanStatus(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).UpdateStatus(plan)
}

func (c *Client) GetPlan(name string, options metav1.GetOptions) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Get(name, options)
}

func (c *Client) ListPlans(options metav1.ListOptions) (*upgradeapiv1.PlanList, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).List(options)
}

func (c *Client) WatchPlans(options metav1.ListOptions) (watch.Interface, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Watch(options)
}

func (c *Client) DeletePlan(name string, options *metav1.DeleteOptions) error {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Delete(name, options)
}

func (c *Client) DeletePlans(options *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).DeleteCollection(options, listOpts)
}

func (c *Client) PollPlanCondition(name string, cond condition.Cond, interval, timeout time.Duration) (plan *upgradeapiv1.Plan, err error) {
	return plan, wait.Poll(interval, timeout, func() (bool, error) {
		plan, err = c.GetPlan(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return cond.GetStatus(plan) != "", nil
	})
}

func (c *Client) BeforeEach() {
	c.beforeFramework()
	c.Framework.BeforeEach()
	c.setupController()
	c.setupChannelServer()
}

func (c *Client) AfterEach() {
	c.teardownChannelServer()
	c.Framework.AfterEach()
}

func (c *Client) setupChannelServer() {
	hostname, err := os.Hostname()
	if err != nil {
		Failf("cannot read hostname: %v", err)
	}
	c.channelServer = &httptest.Server{
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/local/channel/test")
			w.WriteHeader(http.StatusFound)
		})},
	}
	c.channelServer.Listener, err = net.Listen("tcp", net.JoinHostPort(hostname, "0"))
	if err != nil {
		Failf("cannot create tcp listener: %v", err)
	}
	c.channelServer.Start()
}

func (c *Client) teardownChannelServer() {
	c.channelServer.Close()
}

func (c *Client) setupController() {
	var err error
	c.controllerServiceAccount, err = c.ClientSet.CoreV1().ServiceAccounts(c.Namespace.Name).Create(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace.Name,
			Name:      c.Namespace.Name,
		},
	})
	framework.ExpectNoError(err)

	err = frameworkauth.BindClusterRole(c.ClientSet.RbacV1(), "cluster-admin", c.Namespace.Name, rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      c.controllerServiceAccount.Name,
		Namespace: c.controllerServiceAccount.Namespace,
	})
	if err != nil {
		Warnf("%v", err)
	}
	c.controllerDeployment, err = controller.CreateDeployment(c.ClientSet, c.Namespace.Name,
		controller.NewDeployment(c.Namespace.Name,
			controller.DeploymentDefaultImage(),
			controller.DeploymentDefaultTolerations(),
			controller.DeploymentWithServiceAccountName(c.controllerServiceAccount.Name),
		),
	)
	framework.ExpectNoError(err)
}

// beforeFramework is lifted from framework.Framework.BeforeEach when the client is nil
func (c *Client) beforeFramework() {
	ginkgo.By("Creating a kubernetes client")
	config, err := framework.LoadConfig()
	framework.ExpectNoError(err)
	testDesc := ginkgo.CurrentGinkgoTestDescription()
	if len(testDesc.ComponentTexts) > 0 {
		componentTexts := strings.Join(testDesc.ComponentTexts, " ")
		config.UserAgent = fmt.Sprintf(
			"%v -- %v",
			rest.DefaultKubernetesUserAgent(),
			componentTexts)
	}

	config.QPS = c.Framework.Options.ClientQPS
	config.Burst = c.Framework.Options.ClientBurst
	if c.Framework.Options.GroupVersion != nil {
		config.GroupVersion = c.Framework.Options.GroupVersion
	}
	if framework.TestContext.KubeAPIContentType != "" {
		config.ContentType = framework.TestContext.KubeAPIContentType
	}
	c.Framework.ClientSet, err = kubernetes.NewForConfig(config)
	framework.ExpectNoError(err)
	c.Framework.DynamicClient, err = dynamic.NewForConfig(config)
	framework.ExpectNoError(err)
	// node.k8s.io is based on CRD, which is served only as JSON
	jsonConfig := config
	jsonConfig.ContentType = "application/json"
	framework.ExpectNoError(err)

	// create scales getter, set GroupVersion and NegotiatedSerializer to default values
	// as they are required when creating a REST client.
	if config.GroupVersion == nil {
		config.GroupVersion = &schema.GroupVersion{}
	}
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = upgradescheme.Codecs
	}

	c.UpgradeClientSet, err = upgrade.NewForConfig(config)
	framework.ExpectNoError(err)

	restClient, err := rest.RESTClientFor(config)
	framework.ExpectNoError(err)
	discoClient, err := discovery.NewDiscoveryClientForConfig(config)
	framework.ExpectNoError(err)
	cachedDiscoClient := memory.NewMemCacheClient(discoClient)
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoClient)
	restMapper.Reset()
	resolver := scale.NewDiscoveryScaleKindResolver(cachedDiscoClient)
	c.Framework.ScalesGetter = scale.New(restClient, restMapper, dynamic.LegacyAPIPathResolverFunc, resolver)

	framework.TestContext.CloudConfig.Provider.FrameworkBeforeEach(&c.Framework)
}
