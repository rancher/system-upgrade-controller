package job

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/rancher/system-upgrade-controller/pkg/apis/condition"
	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	upgradectr "github.com/rancher/system-upgrade-controller/pkg/upgrade/container"
	upgradenode "github.com/rancher/system-upgrade-controller/pkg/upgrade/node"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	defaultBackoffLimit            = int32(2)
	defaultActiveDeadlineSeconds   = int64(600)
	defaultPrivileged              = true
	defaultKubectlImage            = "rancher/kubectl:v1.21.9"
	defaultImagePullPolicy         = corev1.PullIfNotPresent
	defaultTTLSecondsAfterFinished = int32(900)
)

var (
	ActiveDeadlineSeconds = func(defaultValue int64) int64 {
		if str, ok := os.LookupEnv("SYSTEM_UPGRADE_JOB_ACTIVE_DEADLINE_SECONDS"); ok {
			if i, err := strconv.ParseInt(str, 10, 64); err != nil {
				logrus.Errorf("failed to parse $%s: %v", "SYSTEM_UPGRADE_JOB_ACTIVE_DEADLINE_SECONDS", err)
			} else {
				return i
			}
		}
		return defaultValue
	}(defaultActiveDeadlineSeconds)

	BackoffLimit = func(defaultValue int32) int32 {
		if str, ok := os.LookupEnv("SYSTEM_UPGRADE_JOB_BACKOFF_LIMIT"); ok {
			if i, err := strconv.ParseInt(str, 10, 32); err != nil {
				logrus.Errorf("failed to parse $%s: %v", "SYSTEM_UPGRADE_JOB_BACKOFF_LIMIT", err)
			} else {
				return int32(i)
			}
		}
		return defaultValue
	}(defaultBackoffLimit)

	KubectlImage = func(defaultValue string) string {
		if str := os.Getenv("SYSTEM_UPGRADE_JOB_KUBECTL_IMAGE"); str != "" {
			return str
		}
		return defaultValue
	}(defaultKubectlImage)

	Privileged = func(defaultValue bool) bool {
		if str, ok := os.LookupEnv("SYSTEM_UPGRADE_JOB_PRIVILEGED"); ok {
			if b, err := strconv.ParseBool(str); err != nil {
				logrus.Errorf("failed to parse $%s: %v", "SYSTEM_UPGRADE_JOB_PRIVILEGED", err)
			} else {
				return b
			}
		}
		return defaultValue
	}(defaultPrivileged)

	ImagePullPolicy = func(defaultValue corev1.PullPolicy) corev1.PullPolicy {
		if str := os.Getenv("SYSTEM_UPGRADE_JOB_IMAGE_PULL_POLICY"); str != "" {
			return corev1.PullPolicy(str)
		}
		return defaultValue
	}(defaultImagePullPolicy)

	TTLSecondsAfterFinished = func(defaultValue int32) int32 {
		if str, ok := os.LookupEnv("SYSTEM_UPGRADE_JOB_TTL_SECONDS_AFTER_FINISH"); ok {
			if i, err := strconv.ParseInt(str, 10, 32); err != nil {
				logrus.Errorf("failed to parse $%s: %v", "SYSTEM_UPGRADE_JOB_TTL_SECONDS_AFTER_FINISH", err)
			} else {
				return int32(i)
			}
		}
		return defaultValue
	}(defaultTTLSecondsAfterFinished)
)

var (
	ConditionComplete = condition.Cond(batchv1.JobComplete)
	ConditionFailed   = condition.Cond(batchv1.JobFailed)
)

func New(plan *upgradeapiv1.Plan, node *corev1.Node, controllerName string) *batchv1.Job {
	hostPathDirectory := corev1.HostPathDirectory
	labelPlanName := upgradeapi.LabelPlanName(plan.Name)
	nodeHostname := upgradenode.Hostname(node)
	shortNodeName := strings.SplitN(nodeHostname, ".", 2)[0]
	ownerRefController := true
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.SafeConcatName("apply", plan.Name, "on", shortNodeName, "with", plan.Status.LatestHash),
			Namespace: plan.Namespace,
			Annotations: labels.Set{
				upgradeapi.AnnotationTTLSecondsAfterFinished: strconv.FormatInt(int64(TTLSecondsAfterFinished), 10),
			},
			Labels: labels.Set{
				upgradeapi.LabelController: controllerName,
				upgradeapi.LabelNode:       node.Name,
				upgradeapi.LabelPlan:       plan.Name,
				upgradeapi.LabelVersion:    plan.Status.LatestVersion,
				labelPlanName:              plan.Status.LatestHash,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: plan.APIVersion,
					Kind:       plan.Kind,
					Name:       plan.Name,
					UID:        plan.UID,
					Controller: &ownerRefController,
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &BackoffLimit,
			TTLSecondsAfterFinished: &TTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.Set{
						upgradeapi.LabelController: controllerName,
						upgradeapi.LabelNode:       node.Name,
						upgradeapi.LabelPlan:       plan.Name,
						upgradeapi.LabelVersion:    plan.Status.LatestVersion,
						labelPlanName:              plan.Status.LatestHash,
					},
				},
				Spec: corev1.PodSpec{
					HostIPC:            true,
					HostPID:            true,
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					ServiceAccountName: plan.Spec.ServiceAccountName,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      corev1.LabelHostname,
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											nodeHostname,
										},
									}},
								}},
							},
						},
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{{
										Key:      upgradeapi.LabelPlan,
										Operator: metav1.LabelSelectorOpIn,
										Values: []string{
											plan.Name,
										},
									}},
								},
								TopologyKey: corev1.LabelHostname,
							}},
						},
					},
					Tolerations: append([]corev1.Toleration{{
						Key:      corev1.TaintNodeUnschedulable,
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					}}, plan.Spec.Tolerations...),
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{{
						Name: `host-root`,
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/", Type: &hostPathDirectory,
							},
						},
					}, {
						Name: "pod-info",
						VolumeSource: corev1.VolumeSource{
							DownwardAPI: &corev1.DownwardAPIVolumeSource{
								DefaultMode: &[]int32{420}[0],
								Items: []corev1.DownwardAPIVolumeFile{{
									Path: "labels", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels", APIVersion: "v1"},
								}, {
									Path: "annotations", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations", APIVersion: "v1"},
								}},
							},
						},
					}},
				},
			},
			Completions: new(int32),
			Parallelism: new(int32),
		},
	}

	*job.Spec.Completions = 1
	if i := sort.SearchStrings(plan.Status.Applying, nodeHostname); i < len(plan.Status.Applying) && plan.Status.Applying[i] == nodeHostname {
		*job.Spec.Parallelism = 1
	}

	podTemplate := &job.Spec.Template
	// setup secrets volumes
	for _, secret := range plan.Spec.Secrets {
		podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
			Name: name.SafeConcatName("secret", secret.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		})
	}

	// add volumes from upgrade plan
	for _, v := range plan.Spec.Upgrade.Volumes {
		podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
			Name: v.Name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: v.Source,
				},
			},
		})
	}

	// first, we prepare
	if plan.Spec.Prepare != nil {
		podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers,
			upgradectr.New("prepare", *plan.Spec.Prepare,
				upgradectr.WithLatestTag(plan.Status.LatestVersion),
				upgradectr.WithSecrets(plan.Spec.Secrets),
				upgradectr.WithPlanEnvironment(plan.Name, plan.Status),
				upgradectr.WithImagePullPolicy(ImagePullPolicy),
				upgradectr.WithVolumes(plan.Spec.Upgrade.Volumes),
			),
		)
	}

	// then we cordon/drain
	cordon, drain := plan.Spec.Cordon, plan.Spec.Drain
	if drain != nil {
		args := []string{"drain", node.Name, "--pod-selector", `!` + upgradeapi.LabelController}
		if drain.IgnoreDaemonSets == nil || *plan.Spec.Drain.IgnoreDaemonSets {
			args = append(args, "--ignore-daemonsets")
		}
		if drain.DeleteLocalData == nil || *drain.DeleteLocalData {
			args = append(args, "--delete-local-data")
		}
		if drain.Force {
			args = append(args, "--force")
		}
		if drain.Timeout != nil {
			args = append(args, "--timeout", drain.Timeout.String())
		}
		if drain.GracePeriod != nil {
			args = append(args, "--grace-period", strconv.FormatInt(int64(*drain.GracePeriod), 10))
		}
		if drain.DisableEviction {
			//only available in kubectl version 1.18 or later
			args = append(args, "--disable-eviction=true")
		}
		if drain.SkipWaitForDeleteTimeout > 0 {
			//only available in kubectl version 1.18 or later
			args = append(args, "--skip-wait-for-delete-timeout", strconv.FormatInt(int64(drain.SkipWaitForDeleteTimeout), 10))
		}

		podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers,
			upgradectr.New("drain", upgradeapiv1.ContainerSpec{
				Image: KubectlImage,
				Args:  args,
			},
				upgradectr.WithSecrets(plan.Spec.Secrets),
				upgradectr.WithPlanEnvironment(plan.Name, plan.Status),
				upgradectr.WithImagePullPolicy(ImagePullPolicy),
				upgradectr.WithVolumes(plan.Spec.Upgrade.Volumes),
			),
		)
	} else if cordon {
		podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers,
			upgradectr.New("cordon", upgradeapiv1.ContainerSpec{
				Image: KubectlImage,
				Args:  []string{"cordon", node.Name},
			},
				upgradectr.WithSecrets(plan.Spec.Secrets),
				upgradectr.WithPlanEnvironment(plan.Name, plan.Status),
				upgradectr.WithImagePullPolicy(ImagePullPolicy),
				upgradectr.WithVolumes(plan.Spec.Upgrade.Volumes),
			),
		)
	}

	// and finally, we upgrade
	podTemplate.Spec.Containers = []corev1.Container{
		upgradectr.New("upgrade", *plan.Spec.Upgrade,
			upgradectr.WithLatestTag(plan.Status.LatestVersion),
			upgradectr.WithSecurityContext(&corev1.SecurityContext{
				Privileged: &Privileged,
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						corev1.Capability("CAP_SYS_BOOT"),
					},
				},
			}),
			upgradectr.WithSecrets(plan.Spec.Secrets),
			upgradectr.WithPlanEnvironment(plan.Name, plan.Status),
			upgradectr.WithImagePullPolicy(ImagePullPolicy),
			upgradectr.WithVolumes(plan.Spec.Upgrade.Volumes),
		),
	}

	if ActiveDeadlineSeconds > 0 {
		job.Spec.ActiveDeadlineSeconds = &ActiveDeadlineSeconds
		if drain != nil && drain.Timeout != nil && drain.Timeout.Milliseconds() > ActiveDeadlineSeconds*1000 {
			logrus.Warnf("drain timeout exceeds active deadline seconds")
		}
	}

	return job
}
