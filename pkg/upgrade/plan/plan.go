package plan

import (
	"context"
	"crypto/sha256"
	"fmt"
	stdhash "hash"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kubereboot/kured/pkg/timewindow"
	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/data"
	corectlv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/merr"
	"github.com/rancher/wrangler/v3/pkg/schemas/openapi"
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
	ErrDrainDeleteConflict           = fmt.Errorf("spec.drain cannot specify both deleteEmptydirData and deleteLocalData")
	ErrDrainPodSelectorNotSelectable = fmt.Errorf("spec.drain.podSelector is not selectable")
	ErrInvalidWindow                 = fmt.Errorf("spec.window is invalid")
	ErrInvalidDelay                  = fmt.Errorf("spec.postCompleteDelay is negative")

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
	plan := crd.CRD{
		GVK:        prototype.GroupVersionKind(),
		PluralName: upgradeapiv1.PlanResourceName,
		Status:     true,
		Schema:     schema,
		Categories: []string{"upgrade"},
	}.
		WithColumn("Image", ".spec.upgrade.image").
		WithColumn("Channel", ".spec.channel").
		WithColumn("Version", ".spec.version")
	return &plan, nil
}

func DigestStatus(plan *upgradeapiv1.Plan, secretCache corectlv1.SecretCache) (upgradeapiv1.PlanStatus, error) {
	if upgradeapiv1.PlanLatestResolved.GetReason(plan) != "Error" {
		h := sha256.New224()
		h.Write([]byte(plan.Status.LatestVersion))
		h.Write([]byte(plan.Spec.ServiceAccountName))
		if err := addToHashFromAnnotation(h, plan); err != nil {
			return plan.Status, err
		}

		for _, s := range plan.Spec.Secrets {
			if !s.IgnoreUpdates {
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
		}
		plan.Status.LatestHash = fmt.Sprintf("%x", h.Sum(nil))
	}
	return plan.Status, nil
}

func addToHashFromAnnotation(h stdhash.Hash, plan *upgradeapiv1.Plan) error {
	if plan.Annotations[upgradeapi.AnnotationIncludeInDigest] == "" {
		return nil
	}

	dataMap, err := data.Convert(plan)
	if err != nil {
		return err
	}

	for _, entry := range strings.Split(plan.Annotations[upgradeapi.AnnotationIncludeInDigest], ",") {
		h.Write([]byte(dataMap.String(strings.Split(entry, ".")...)))
	}

	return nil
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
	if (response.StatusCode / 200) == 1 {
		return filepath.Base(url), nil
	}
	return "", fmt.Errorf("unexpected response: %s %s", response.Proto, response.Status)
}

func NodeSelector(plan *upgradeapiv1.Plan) (labels.Selector, error) {
	nodeSelector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector)
	if err != nil {
		return nil, err
	}
	requireHostname, err := labels.NewRequirement(corev1.LabelHostname, selection.Exists, []string{})
	if err != nil {
		return nil, err
	}
	return nodeSelector.Add(*requireHostname), nil
}

func SelectConcurrentNodes(plan *upgradeapiv1.Plan, nodeCache corectlv1.NodeCache) ([]*corev1.Node, error) {
	var (
		applying = plan.Status.Applying
		selected []*corev1.Node
	)
	nodeSelector, err := NodeSelector(plan)
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
			selected = append(selected, node.DeepCopy())
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

		for i := 0; i < len(candidateNodes) && int64(len(selected)) < plan.Spec.Concurrency; i++ {
			selected = append(selected, candidateNodes[i].DeepCopy())
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Name < selected[j].Name
	})
	return selected, nil
}

func sha256sum(s ...string) string {
	h := sha256.New()
	for i := range s {
		h.Write([]byte(s[i]))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Validate performs validation of the plan spec, raising errors for any conflicting or invalid settings.
func Validate(plan *upgradeapiv1.Plan, secretCache corectlv1.SecretCache) error {
	if drainSpec := plan.Spec.Drain; drainSpec != nil {
		if drainSpec.DeleteEmptydirData != nil && drainSpec.DeleteLocalData != nil {
			return ErrDrainDeleteConflict
		}
		if drainSpec.PodSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(drainSpec.PodSelector)
			if err != nil {
				return err
			}
			if _, ok := selector.Requirements(); !ok {
				return ErrDrainPodSelectorNotSelectable
			}
		}
	}
	if windowSpec := plan.Spec.Window; windowSpec != nil {
		if _, err := timewindow.New(windowSpec.Days, windowSpec.StartTime, windowSpec.EndTime, windowSpec.TimeZone); err != nil {
			return merr.NewErrors(ErrInvalidWindow, err)
		}
	}
	if delay := plan.Spec.PostCompleteDelay; delay != nil && delay.Duration < 0 {
		return ErrInvalidDelay
	}

	sErrs := []error{}
	for _, secret := range plan.Spec.Secrets {
		if secret.IgnoreUpdates {
			continue
		}
		if _, err := secretCache.Get(plan.Namespace, secret.Name); err != nil {
			sErrs = append(sErrs, err)
		}
	}

	return merr.NewErrors(sErrs...)
}
