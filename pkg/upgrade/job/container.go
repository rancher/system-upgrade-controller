package job

import (
	"path/filepath"
	"strings"

	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
)

type containerOpt func(*corev1.Container)

func withSecrets(secrets []upgradeapiv1.SecretSpec) containerOpt {
	return func(container *corev1.Container) {
		for _, secret := range secrets {
			secretVolumeName := name.SafeConcatName("secret", secret.Name)
			secretVolumePath := secret.Path
			if secretVolumePath == "" {
				secretVolumePath = filepath.Join("/run/system-upgrade/secrets", secret.Name)
			} else if secretVolumePath[0:1] != "/" {
				secretVolumePath = filepath.Join("/run/system-upgrade/secrets", secretVolumePath)
			}
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      secretVolumeName,
				MountPath: secretVolumePath,
				ReadOnly:  true,
			})
		}
	}
}

func withSecurityContext(securityContext *corev1.SecurityContext) containerOpt {
	return func(container *corev1.Container) {
		container.SecurityContext = securityContext
	}
}

func withImageTag(tag string) containerOpt {
	return func(container *corev1.Container) {
		image := container.Image
		if p := strings.Split(image, `:`); len(p) > 1 {
			image = strings.Join(p[0:len(p)-1], `:`)
		}
		container.Image = image + `:` + tag
	}
}

func withPlanEnvironment(planName string, planStatus upgradeapiv1.PlanStatus) containerOpt {
	return func(container *corev1.Container) {
		container.Env = append(container.Env, []corev1.EnvVar{{
			Name:  "SYSTEM_UPGRADE_PLAN_NAME",
			Value: planName,
		}, {
			Name:  "SYSTEM_UPGRADE_PLAN_LATEST_HASH",
			Value: planStatus.LatestHash,
		}, {
			Name:  "SYSTEM_UPGRADE_PLAN_LATEST_VERSION",
			Value: planStatus.LatestVersion,
		}}...)
	}
}

func container(name string, spec upgradeapiv1.ContainerSpec, opt ...containerOpt) corev1.Container {
	container := corev1.Container{
		Name:            name,
		Image:           spec.Image,
		ImagePullPolicy: ImagePullPolicy,
		Command:         spec.Command,
		Args:            spec.Args,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "host-root", MountPath: "/host"},
			{Name: "pod-info", MountPath: "/run/system-upgrade/pod", ReadOnly: true},
		},
		Env: []corev1.EnvVar{{
			Name: "SYSTEM_UPGRADE_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}, {
			Name: "SYSTEM_UPGRADE_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}, {
			Name: "SYSTEM_UPGRADE_POD_UID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		}},
	}
	for _, fn := range opt {
		fn(&container)
	}
	return container
}
