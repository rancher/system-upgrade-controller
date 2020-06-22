package plan

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	corectlv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/schemas/openapi"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/kubectl/pkg/util/hash"
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

func CRD() (*crd.CRD, error) {
	prototype := upgradeapiv1.NewPlan("", "", upgradeapiv1.Plan{})
	schema, err := openapi.ToOpenAPIFromStruct(*prototype)
	if err != nil {
		return nil, err
	}
	return &crd.CRD{
		GVK:        prototype.GroupVersionKind(),
		PluralName: upgradeapiv1.PlanResourceName,
		Status:     true,
		Schema:     schema,
		Categories: []string{"upgrade"},
	}, nil
}

func DigestStatus(plan *upgradeapiv1.Plan, secretCache corectlv1.SecretCache) (upgradeapiv1.PlanStatus, error) {
	if upgradeapiv1.PlanLatestResolved.GetReason(plan) != "Error" {
		h := sha256.New224()
		h.Write([]byte(plan.Status.LatestVersion))
		h.Write([]byte(plan.Spec.ServiceAccountName))
		for _, s := range plan.Spec.Secrets {
			secret, err := secretCache.Get(plan.Namespace, s.Name)
			if err != nil {
				return plan.Status, err
			}
			secretHash, err := hash.SecretHash(secret)
			if err != nil {
				return plan.Status, err
			}
			h.Write([]byte(secretHash))
		}
		plan.Status.LatestHash = fmt.Sprintf("%x", h.Sum(nil))
	}
	return plan.Status, nil
}

func MungeVersion(version string) string {
	return strings.ReplaceAll(version, `+`, `-`)
}

const (
	headerClusterID     = `X-SUC-Cluster-ID`
	headerLatestVersion = `X-SUC-Latest-Version`
)

func ResolveChannel(ctx context.Context, url, latestVersion, clusterID string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	logrus.Debugf("Preparing to resolve %q", url)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if clusterID != "" {
		request.Header[headerClusterID] = []string{clusterID}
	}
	if latestVersion != "" {
		request.Header[headerLatestVersion] = []string{latestVersion}
	}
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
	return filepath.Base(url), nil
}

func SelectConcurrentNodeNames(plan *upgradeapiv1.Plan, nodeCache corectlv1.NodeCache) ([]string, error) {
	var (
		applying = plan.Status.Applying
		selected []string
	)
	nodeSelector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector)
	if err != nil {
		return nil, err
	}
	requirementPlanNotLatest, err := labels.NewRequirement(upgradeapi.LabelPlanName(plan.Name), selection.NotIn, []string{"disabled", plan.Status.LatestHash})
	if err != nil {
		return nil, err
	}
	nodeSelector = nodeSelector.Add(*requirementPlanNotLatest)
	if len(applying) > 0 {
		requirementApplying, err := labels.NewRequirement(corev1.LabelHostname, selection.In, applying)
		if err != nil {
			return nil, err
		}
		applyingNodes, err := nodeCache.List(nodeSelector.Add(*requirementApplying))
		if err != nil {
			return nil, err
		}
		for _, node := range applyingNodes {
			selected = append(selected, node.Name)
		}
		requirementNotApplying, err := labels.NewRequirement(corev1.LabelHostname, selection.NotIn, applying)
		if err != nil {
			return nil, err
		}
		nodeSelector = nodeSelector.Add(*requirementNotApplying)
	}

	// avoid listing, sorting, and appending candidate nodes if we can
	if int64(len(selected)) < plan.Spec.Concurrency {
		candidateNodes, err := nodeCache.List(nodeSelector)
		if err != nil {
			return nil, err
		}
		// this code exists to establish a defined order for generating jobs per plan, per latest hash.
		// it is necessary to avoid the sometimes occurrence of multiple calls to the generating handler for the same
		// plan resource version which, due to undefined ordering when listing nodes, was causing more jobs to be
		// generated than dictated by the plan concurrency
		sort.Slice(candidateNodes, func(i, j int) bool {
			isum := sha256sum(string(candidateNodes[i].UID), string(plan.UID), plan.Status.LatestHash)
			jsum := sha256sum(string(candidateNodes[j].UID), string(plan.UID), plan.Status.LatestHash)
			return isum < jsum
		})
		for _, node := range applying {
			selected = AppendIfMissing(selected, node)
		}
		for i := 0; i < len(candidateNodes) && int64(len(selected)) < plan.Spec.Concurrency; i++ {
			selected = AppendIfMissing(selected, candidateNodes[i].Name)
		}
	}
	sort.Strings(selected)
	return selected, nil
}

func sha256sum(s ...string) string {
	h := sha256.New()
	for i := range s {
		h.Write([]byte(s[i]))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func AppendIfMissing(slice []string, element string) []string {
	for _, ele := range slice {
		if ele == element {
			return slice
		}
	}
	return append(slice, element)
}
