package upgrade

import (
	"context"
	"errors"
	"fmt"
	"time"

	upgradecln "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned"
	upgradectl "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io"
	upgradeinf "github.com/rancher/system-upgrade-controller/pkg/generated/informers/externalversions"
	upgradeplan "github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
	batchctl "github.com/rancher/wrangler-api/pkg/generated/controllers/batch"
	corectl "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/start"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	ErrControllerNameRequired      = errors.New("controller name is required")
	ErrControllerNamespaceRequired = errors.New("controller namespace is required")
)

type Controller struct {
	Namespace string
	Name      string

	cfg *rest.Config
	kcs *kubernetes.Clientset
	ucs *upgradecln.Clientset

	clusterID string

	coreFactory    *corectl.Factory
	batchFactory   *batchctl.Factory
	upgradeFactory *upgradectl.Factory

	apply apply.Apply
}

func NewController(cfg *rest.Config, namespace, name string) (ctl *Controller, err error) {
	if namespace == "" {
		return nil, ErrControllerNamespaceRequired
	}
	if name == "" {
		return nil, ErrControllerNameRequired
	}

	if cfg == nil {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	ctl = &Controller{
		Namespace: namespace,
		Name:      name,
		cfg:       cfg,
	}

	ctl.kcs, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// cluster id hack: see https://groups.google.com/forum/#!msg/kubernetes-sig-architecture/mVGobfD4TpY/nkdbkX1iBwAJ
	systemNS, err := ctl.kcs.CoreV1().Namespaces().Get(metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ctl.clusterID = fmt.Sprintf("%s", systemNS.UID)

	ctl.ucs, err = upgradecln.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	ctl.apply, err = apply.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return ctl, nil
}

func (ctl *Controller) Start(ctx context.Context, threads int, resync time.Duration) error {
	if err := ctl.registerCRD(ctx); err != nil {
		return err
	}

	kubeInformers := informers.NewSharedInformerFactoryWithOptions(ctl.kcs, resync, informers.WithNamespace(ctl.Namespace))
	ctl.coreFactory = corectl.NewFactory(ctl.kcs, kubeInformers)
	ctl.batchFactory = batchctl.NewFactory(ctl.kcs, kubeInformers)

	upgradeInformers := upgradeinf.NewSharedInformerFactoryWithOptions(ctl.ucs, resync, upgradeinf.WithNamespace(ctl.Namespace))
	ctl.upgradeFactory = upgradectl.NewFactory(ctl.ucs, upgradeInformers)

	// register our handlers
	if err := ctl.handleJobs(ctx); err != nil {
		return err
	}
	if err := ctl.handleNodes(ctx); err != nil {
		return err
	}
	if err := ctl.handlePlans(ctx); err != nil {
		return err
	}
	if err := ctl.handleSecrets(ctx); err != nil {
		return err
	}

	return start.All(ctx, threads, ctl.coreFactory, ctl.batchFactory, ctl.upgradeFactory)
}

type crdFunc func() (*crd.CRD, error)

func (ctl *Controller) registerCRD(ctx context.Context) error {
	factory, err := crd.NewFactoryFromClient(ctl.cfg)
	if err != nil {
		return err
	}

	var crds []crd.CRD
	for _, crdFn := range []crdFunc{
		upgradeplan.CRD,
	} {
		crdRef, err := crdFn()
		if err != nil {
			return err
		}
		crds = append(crds, *crdRef)
	}

	return factory.BatchCreateCRDs(ctx, crds...).BatchWait()
}
