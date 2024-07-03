package upgrade

import (
	"context"
	"errors"
	"fmt"
	"time"

	upgradectl "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io"
	upgradeplan "github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/crd"
	batchctl "github.com/rancher/wrangler/pkg/generated/controllers/batch"
	corectl "github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/start"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	clusterID string

	coreFactory    *corectl.Factory
	batchFactory   *batchctl.Factory
	upgradeFactory *upgradectl.Factory

	upgradeWindow *Periodic

	apply apply.Apply
}

func NewController(cfg *rest.Config, namespace, name string, resync time.Duration, windowStart string, windowLength string) (ctl *Controller, err error) {
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

	var upgradeWindow *Periodic

	if windowStart != "" && windowLength != "" {
		uw, err := ParsePeriodic(windowStart, windowLength)
		if err != nil {
			return nil, fmt.Errorf("parsing reboot window: %w", err)
		}

		upgradeWindow = uw
	}

	ctl = &Controller{
		Namespace:     namespace,
		Name:          name,
		cfg:           cfg,
		upgradeWindow: upgradeWindow,
	}

	ctl.kcs, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	ctl.coreFactory, err = corectl.NewFactoryFromConfigWithOptions(cfg, &corectl.FactoryOptions{
		Namespace: namespace,
		Resync:    resync,
	})
	if err != nil {
		return nil, err
	}
	ctl.batchFactory, err = batchctl.NewFactoryFromConfigWithOptions(cfg, &batchctl.FactoryOptions{
		Namespace: namespace,
		Resync:    resync,
	})
	if err != nil {
		return nil, err
	}
	ctl.upgradeFactory, err = upgradectl.NewFactoryFromConfigWithOptions(cfg, &corectl.FactoryOptions{
		Namespace: namespace,
		Resync:    resync,
	})
	if err != nil {
		return nil, err
	}
	ctl.apply, err = apply.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return ctl, nil
}

func (ctl *Controller) Start(ctx context.Context, threads int) error {
	// cluster id hack: see https://groups.google.com/forum/#!msg/kubernetes-sig-architecture/mVGobfD4TpY/nkdbkX1iBwAJ
	systemNS, err := ctl.kcs.CoreV1().Namespaces().Get(ctx, metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ctl.clusterID = fmt.Sprintf("%s", systemNS.UID)

	if err := ctl.registerCRD(ctx); err != nil {
		return err
	}

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

func (ctl *Controller) registerCRD(ctx context.Context) error {
	factory, err := crd.NewFactoryFromClient(ctl.cfg)
	if err != nil {
		return err
	}

	var crds []crd.CRD
	for _, crdFn := range []func() (*crd.CRD, error){
		upgradeplan.CRD,
	} {
		crdef, err := crdFn()
		if err != nil {
			return err
		}
		crds = append(crds, *crdef)
	}

	return factory.BatchCreateCRDs(ctx, crds...).BatchWait()
}

// insideUpgradeWindow checks if process is inside reboot window at the time
// of calling this function.
//
// If reboot window is not configured, true is always returned.
func (ctl *Controller) insideUpgradeWindow() bool {
	if ctl.upgradeWindow == nil {
		return true
	}

	// Most recent reboot window might still be open.
	mostRecentRebootWindow := ctl.upgradeWindow.Previous(time.Now())

	return time.Now().Before(mostRecentRebootWindow.End)
}
