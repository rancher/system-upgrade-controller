package job

import (
	"os"
	"strconv"

	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
)

const (
	defaultBackoffLimit          = int32(2)
	defaultActiveDeadlineSeconds = int64(600)
	defaultPrivileged            = true
	defaultKubectlImage          = "rancher/kubectl:latest"
	defaultImagePullPolicy       = corev1.PullIfNotPresent
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
)

func NewUpgradeJob(plan *upgradeapiv1.Plan, nodeName, controllerName string) *batchv1.Job {
	labelPlanName := upgradeapi.LabelPlanName(plan.Name)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.SafeConcatName("upgrade", nodeName, "with", plan.Name, "at", plan.Status.LatestHash),
			Namespace: plan.Namespace,
			Labels: labels.Set{
				upgradeapi.LabelController: controllerName,
				upgradeapi.LabelNode:       nodeName,
				upgradeapi.LabelPlan:       plan.Name,
				labelPlanName:              plan.Status.LatestHash,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &BackoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.Set{
						upgradeapi.LabelController: controllerName,
						upgradeapi.LabelNode:       nodeName,
						upgradeapi.LabelPlan:       plan.Name,
						labelPlanName:              plan.Status.LatestHash,
					},
				},
				Spec: corev1.PodSpec{
					HostIPC:            true,
					HostPID:            true,
					HostNetwork:        true,
					ServiceAccountName: plan.Spec.ServiceAccountName,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      corev1.LabelHostname,
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											nodeName,
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
					Tolerations: []corev1.Toleration{{
						Key:      schedulerapi.TaintNodeUnschedulable,
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					}},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       volumes(plan.Spec.Secrets),
					Containers: []corev1.Container{
						container("upgrade", *plan.Spec.Upgrade,
							withImageTag(plan.Status.LatestVersion),
							withSecurityContext(&corev1.SecurityContext{
								Privileged: &Privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										corev1.Capability("CAP_SYS_BOOT"),
									},
								},
							}),
							withSecrets(plan.Spec.Secrets),
							withPlanEnvironment(plan.Name, plan.Status),
						),
					},
				},
			},
		},
	}

	// first, we prepare
	if plan.Spec.Prepare != nil {
		job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers,
			container("prepare", *plan.Spec.Prepare,
				withSecrets(plan.Spec.Secrets),
				withPlanEnvironment(plan.Name, plan.Status),
			),
		)
	}

	// then we cordon/drain
	cordon, drain := plan.Spec.Cordon, plan.Spec.Drain
	if drain != nil {
		args := []string{"drain", nodeName, "--pod-selector", `!` + upgradeapi.LabelController}
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
		job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers,
			container("drain", upgradeapiv1.ContainerSpec{
				Image: KubectlImage,
				Args:  args,
			},
				withSecrets(plan.Spec.Secrets),
				withPlanEnvironment(plan.Name, plan.Status),
			),
		)
	} else if cordon {
		job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers,
			container("cordon", upgradeapiv1.ContainerSpec{
				Image: KubectlImage,
				Args:  []string{"cordon", nodeName},
			},
				withSecrets(plan.Spec.Secrets),
				withPlanEnvironment(plan.Name, plan.Status),
			),
		)
	}

	if ActiveDeadlineSeconds > 0 {
		job.Spec.ActiveDeadlineSeconds = &ActiveDeadlineSeconds
		if drain != nil && drain.Timeout != nil && drain.Timeout.Milliseconds() > ActiveDeadlineSeconds*1000 {
			logrus.Warnf("drain timeout exceeds active deadline seconds")
		}
	}

	return job
}

func volumes(secrets []upgradeapiv1.SecretSpec) []corev1.Volume {
	hostPathDirectory := corev1.HostPathDirectory
	volumes := []corev1.Volume{{
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
				Items: []corev1.DownwardAPIVolumeFile{{
					Path: "labels",
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels",
					},
				}, {
					Path: "annotations",
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.annotations",
					},
				}},
			},
		},
	}}
	for _, secret := range secrets {
		volumes = append(volumes, corev1.Volume{
			Name: name.SafeConcatName("secret", secret.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		})
	}
	return volumes
}
