package upgrade

import (
	"context"

	upgradectl "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io"
	"github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
	batchctl "github.com/rancher/wrangler-api/pkg/generated/controllers/batch"
	corectl "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/start"
	"k8s.io/client-go/rest"
)

// StartController starts the controller
func StartController(ctx context.Context, config *rest.Config, threads int, namespace, serviceAccountName, name string) error {
	if crdFactory, err := crd.NewFactoryFromClient(config); err != nil {
		return err
	} else if err = plan.RegisterCRD(ctx, crdFactory); err != nil {
		return err
	} else if err = crdFactory.BatchWait(); err != nil {
		return err
	}

	coreFactory, err := corectl.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		return err
	}
	batchFactory, err := batchctl.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		return err
	}
	upgradeFactory, err := upgradectl.NewFactoryFromConfigWithNamespace(config, namespace)
	if err != nil {
		return err
	}

	apply, err := apply.NewForConfig(config)
	if err != nil {
		return err
	}

	err = plan.RegisterHandlers(ctx, serviceAccountName, namespace, name, apply, upgradeFactory, coreFactory, batchFactory)
	if err != nil {
		return err
	}

	return start.All(ctx, threads, coreFactory, batchFactory, upgradeFactory)
}
