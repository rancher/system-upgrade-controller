package framework

import (
	"context"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/rancher/system-upgrade-controller/e2e/framework/controller"
	"github.com/rancher/system-upgrade-controller/pkg/apis/condition"
	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradecln "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned"
	upgradescheme "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/scheme"
	upgradejob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	admissionapi "k8s.io/pod-security-admission/api"
)

type Option func(*Options)

type Options struct {
	framework.Options
}

type Client struct {
	framework.Framework

	UpgradeClientSet *upgradecln.Clientset

	controllerDeployment     *appsv1.Deployment
	controllerServiceAccount *corev1.ServiceAccount
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
		Framework: *framework.NewFramework(name, options.Options, nil),
	}
	client.Framework.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	ginkgo.BeforeEach(client.BeforeEach)
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

func (c *Client) CreatePlan(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Create(context.TODO(), plan, metav1.CreateOptions{})
}

func (c *Client) UpdatePlan(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Update(context.TODO(), plan, metav1.UpdateOptions{})
}

func (c *Client) UpdatePlanStatus(plan *upgradeapiv1.Plan) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).UpdateStatus(context.TODO(), plan, metav1.UpdateOptions{})
}

func (c *Client) GetPlan(name string, options metav1.GetOptions) (*upgradeapiv1.Plan, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Get(context.TODO(), name, options)
}

func (c *Client) ListPlans(options metav1.ListOptions) (*upgradeapiv1.PlanList, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).List(context.TODO(), options)
}

func (c *Client) WatchPlans(options metav1.ListOptions) (watch.Interface, error) {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Watch(context.TODO(), options)
}

func (c *Client) DeletePlan(name string, options metav1.DeleteOptions) error {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).Delete(context.TODO(), name, options)
}

func (c *Client) DeletePlans(options metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return c.UpgradeClientSet.UpgradeV1().Plans(c.Namespace.Name).DeleteCollection(context.TODO(), options, listOpts)
}

func (c *Client) CreateSecret(secret *corev1.Secret) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(c.Namespace.Name).Create(context.TODO(), secret, metav1.CreateOptions{})
}

func (c *Client) UpdateSecret(secret *corev1.Secret) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(c.Namespace.Name).Update(context.TODO(), secret, metav1.UpdateOptions{})
}

func (c *Client) WaitForPlanCondition(name string, cond condition.Cond, timeout time.Duration) (plan *upgradeapiv1.Plan, err error) {
	return plan, wait.Poll(time.Second, timeout, func() (bool, error) {
		plan, err = c.GetPlan(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return cond.GetStatus(plan) != "", nil
	})
}

func (c *Client) WaitForPlanJobs(plan *upgradeapiv1.Plan, count int, timeout time.Duration) (jobs []batchv1.Job, err error) {
	labelSelector := labels.SelectorFromSet(labels.Set{
		upgradeapi.LabelPlan: plan.Name,
	})

	return jobs, wait.Poll(5*time.Second, timeout, func() (bool, error) {
		list, err := c.ClientSet.BatchV1().Jobs(plan.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			return false, err
		}
		for _, item := range list.Items {
			if upgradejob.ConditionFailed.IsTrue(&item) || upgradejob.ConditionComplete.IsTrue(&item) {
				jobs = append(jobs, item)
			}
		}
		return len(jobs) >= count, nil
	})
}

func (c *Client) BeforeEach(ctx context.Context) {
	c.beforeFramework()
	c.Framework.BeforeEach(ctx)
	c.setupController()
}

func (c *Client) AfterEach(ctx context.Context) {
	c.Framework.AfterEach(ctx)
}

func (c *Client) setupController() {
	var err error
	c.controllerServiceAccount, err = c.ClientSet.CoreV1().ServiceAccounts(c.Namespace.Name).Create(context.TODO(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace.Name,
			Name:      c.Namespace.Name,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	_, err = frameworkauth.BindClusterRole(context.TODO(), c.ClientSet.RbacV1(), "cluster-admin", c.Namespace.Name, rbacv1.Subject{
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

	c.UpgradeClientSet, err = upgradecln.NewForConfig(config)
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
