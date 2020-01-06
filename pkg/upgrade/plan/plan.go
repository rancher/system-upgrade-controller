package plan

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rancher/norman/pkg/openapi"
	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradectl "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io"
	upgradectlv1 "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io/v1"
	jobctl "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	batchctl "github.com/rancher/wrangler-api/pkg/generated/controllers/batch"
	corectl "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	corectlv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	defaultPollingInterval = 15 * time.Minute
)

var (
	PollingInterval = func(defaultValue time.Duration) time.Duration {
		if str, ok := os.LookupEnv("SYSTEM_UPGRADE_PLAN_POLLING_INTERVAL"); ok {
			if d, err := time.ParseDuration(str); err != nil {
				logrus.Errorf("failed to parse $%s: %v", "SYSTEM_UPGRADE_PLAN_POLLING_INTERVAL", err)
			} else if d > time.Minute {
				return d
			}
		}
		return defaultValue
	}(defaultPollingInterval)
)

// RegisterCRD registers the Plan custom resource definition
func RegisterCRD(ctx context.Context, factory *crd.Factory) error {
	prototype := upgradeapiv1.NewPlan("", "", upgradeapiv1.Plan{})
	schema, err := openapi.ToOpenAPIFromStruct(*prototype)
	if err != nil {
		return err
	}
	factory.BatchCreateCRDs(ctx, crd.CRD{
		GVK:        prototype.GroupVersionKind(),
		PluralName: upgradeapiv1.PlanResourceName,
		Status:     true,
		Schema:     schema,
		Categories: []string{"upgrade"},
	})
	return nil
}

// RegisterHandlers registers Plan handlers
func RegisterHandlers(ctx context.Context, serviceAccountName, controllerNamespace, controllerName string, apply apply.Apply, upgradeFactory *upgradectl.Factory, coreFactory *corectl.Factory, batchFactory *batchctl.Factory) error {
	plans := upgradeFactory.Upgrade().V1().Plan()
	jobs := batchFactory.Batch().V1().Job()
	nodes := coreFactory.Core().V1().Node()

	// cluster id hack: see https://groups.google.com/forum/#!msg/kubernetes-sig-architecture/mVGobfD4TpY/nkdbkX1iBwAJ
	systemNS, err := coreFactory.Core().V1().Namespace().Get(metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nodes.OnChange(ctx, controllerName, func(key string, obj *corev1.Node) (*corev1.Node, error) {
		if obj == nil {
			return obj, nil
		}
		if planList, err := plans.Cache().List(controllerNamespace, labels.Everything()); err != nil {
			logrus.Error(err)
		} else {
			for _, plan := range planList {
				if selector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector); err != nil {
					logrus.Error(err)
				} else if selector.Matches(labels.Set(obj.Labels)) {
					plans.Enqueue(plan.Namespace, plan.Name)
				}
			}
		}
		return obj, nil
	})

	jobs.OnChange(ctx, controllerName, func(key string, obj *batchv1.Job) (*batchv1.Job, error) {
		if obj == nil {
			return obj, nil
		}
		if obj.Labels != nil {
			if planName, ok := obj.Labels[upgradeapi.LabelPlan]; ok {
				defer plans.Enqueue(obj.Namespace, planName)
				if obj.Status.Succeeded == 1 {
					planLabel := upgradeapi.LabelPlanName(planName)
					if version, ok := obj.Labels[planLabel]; ok {
						if nodeName, ok := obj.Labels[upgradeapi.LabelNode]; ok {
							node, err := nodes.Get(nodeName, metav1.GetOptions{})
							if err != nil {
								return obj, err
							}
							node.Labels[planLabel] = version
							if node, err = nodes.Update(node); err != nil {
								return obj, err
							}
						}
					}
				}
			}
		}
		return obj, nil
	})

	upgradectlv1.RegisterPlanStatusHandler(ctx, plans, "", controllerName,
		func(obj *upgradeapiv1.Plan, status upgradeapiv1.PlanStatus) (upgradeapiv1.PlanStatus, error) {
			resolved := upgradeapiv1.PlanLatestResolved
			resolved.CreateUnknownIfNotExists(obj)
			if obj.Spec.Version == "" && obj.Spec.Channel == "" {
				resolved.SetError(obj, "Error", fmt.Errorf("missing one of channel or version"))
				return obj.Status, nil
			}
			if obj.Spec.Version != "" {
				resolved.SetError(obj, "Version", nil)
				obj.Status.LatestVersion = obj.Spec.Version
				return obj.Status, nil
			}
			if resolved.IsTrue(obj) {
				if lastUpdated, err := time.Parse(time.RFC3339, resolved.GetLastUpdated(obj)); err == nil {
					if interval := time.Now().Sub(lastUpdated); interval < PollingInterval {
						plans.EnqueueAfter(obj.Namespace, obj.Name, PollingInterval-interval)
						return status, nil
					}
				}
			}
			latest, err := resolveChannel(ctx, obj.Spec.Channel, string(systemNS.UID))
			if err != nil {
				return status, err
			}
			resolved.SetError(obj, "Channel", nil)
			obj.Status.LatestVersion = latest
			return obj.Status, nil
		},
	)

	upgradectlv1.RegisterPlanGeneratingHandler(ctx, plans, apply.WithCacheTypes(jobs).WithCacheTypes(nodes).WithNoDelete(), "", controllerName,
		func(obj *upgradeapiv1.Plan, status upgradeapiv1.PlanStatus) (objects []runtime.Object, _ upgradeapiv1.PlanStatus, _ error) {
			concurrentNodeNames, err := selectConcurrentNodeNames(nodes.Cache(), obj)
			if err != nil {
				logrus.Error(err)
				return objects, status, nil
			}
			logrus.Debugf("concurrentNodeNames = %q", concurrentNodeNames)
			for _, nodeName := range concurrentNodeNames {
				objects = append(objects, jobctl.NewUpgradeJob(obj, serviceAccountName, nodeName, controllerName))
			}
			obj.Status.Applying = concurrentNodeNames
			logrus.Debugf("%#v", objects)
			return objects, obj.Status, nil
		},
		&generic.GeneratingHandlerOptions{
			AllowClusterScoped: true,
		},
	)
	return nil
}

func resolveChannel(ctx context.Context, channelURL, clusterID string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	logrus.Debugf("Preparing to resolve %q", channelURL)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, channelURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set(`x-`+metav1.NamespaceSystem, string(clusterID))
	logrus.Debugf("Sending %+v", request)
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusFound {
		redirect, err := response.Location()
		if err != nil {
			return "", err
		}
		return filepath.Base(redirect.Path), nil
	}
	return filepath.Base(channelURL), nil
}

func selectConcurrentNodeNames(nodeCache corectlv1.NodeCache, plan *upgradeapiv1.Plan) ([]string, error) {
	var (
		applying = plan.Status.Applying
		selected []string
	)
	logrus.Debugf("plan.spec.nodeSelector = %+v", plan.Spec.NodeSelector)
	nodeSelector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector)
	if err != nil {
		return nil, err
	}
	requirementPlanNotLatest, err := labels.NewRequirement(upgradeapi.LabelPlanName(plan.Name), selection.NotIn, []string{"disabled", plan.Status.LatestVersion})
	if err != nil {
		return nil, err
	}
	nodeSelector = nodeSelector.Add(*requirementPlanNotLatest)
	logrus.Debugf("nodeSelector = %+v", nodeSelector)
	if len(applying) > 0 {
		requirementApplying, err := labels.NewRequirement(corev1.LabelHostname, selection.In, applying)
		if err != nil {
			return nil, err
		}
		applyingNodes, err := nodeCache.List(nodeSelector.Add(*requirementApplying))
		if err != nil {
			return nil, err
		}
		logrus.Debugf("applyingNodes = %+v", applyingNodes)
		for _, node := range applyingNodes {
			selected = append(selected, node.Name)
		}
		requirementNotApplying, err := labels.NewRequirement(corev1.LabelHostname, selection.NotIn, applying)
		if err != nil {
			return nil, err
		}
		nodeSelector = nodeSelector.Add(*requirementNotApplying)
	}

	candidateNodes, err := nodeCache.List(nodeSelector)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("candidateNodes = %+v", candidateNodes)
	for i := 0; i < len(candidateNodes) && int64(len(selected)) < plan.Spec.Concurrency; i++ {
		selected = append(selected, candidateNodes[i].Name)
	}

	sort.Strings(selected)
	return selected, nil
}
