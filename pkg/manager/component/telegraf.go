package component

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"yunion.io/x/onecloud-operator/pkg/apis/constants"

	"yunion.io/x/onecloud-operator/pkg/apis/onecloud/v1alpha1"
	"yunion.io/x/onecloud-operator/pkg/manager"
)

type telegrafManager struct {
	*ComponentManager
}

func newTelegrafManager(man *ComponentManager) manager.Manager {
	return &telegrafManager{ComponentManager: man}
}

func (m *telegrafManager) Sync(oc *v1alpha1.OnecloudCluster) error {
	return syncComponent(m, oc, oc.Spec.Influxdb.Disable)
}

func (m *telegrafManager) getDaemonSet(
	oc *v1alpha1.OnecloudCluster,
	cfg *v1alpha1.OnecloudClusterConfig,
) (*apps.DaemonSet, error) {
	return m.newTelegrafDaemonSet(v1alpha1.TelegrafComponentType, oc, cfg)
}

func (m *telegrafManager) newTelegrafDaemonSet(
	cType v1alpha1.ComponentType,
	oc *v1alpha1.OnecloudCluster,
	cfg *v1alpha1.OnecloudClusterConfig,
) (*apps.DaemonSet, error) {
	dsSpec := oc.Spec.Telegraf
	containersF := func(volMounts []corev1.VolumeMount) []corev1.Container {
		return []corev1.Container{
			{
				Name:            cType.String(),
				Image:           dsSpec.Image, // TODO: set default_image
				ImagePullPolicy: dsSpec.ImagePullPolicy,
				Command: []string{
					"/usr/bin/telegraf",
					"-config", "/etc/telegraf/telegraf.conf",
					"-config-directory", "/etc/telegraf/telegraf.d",
				},
				VolumeMounts: volMounts,
			},
		}
	}
	initContainers := func(volMounts []corev1.VolumeMount) []corev1.Container {
		return []corev1.Container{
			{
				Name:            cType.String() + "-init",
				Image:           dsSpec.InitContainerImage,
				ImagePullPolicy: dsSpec.ImagePullPolicy,
				Command:         []string{"/bin/telegraf-init"},
				VolumeMounts:    volMounts,
				Env: []corev1.EnvVar{
					{
						Name: "NODENAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						},
					},
					{
						Name:  "INFLUXDB_URL",
						Value: getInfluxDBInternalURL(oc),
					},
				},
			},
		}
	}([]corev1.VolumeMount{{
		Name:      "etc-telegraf",
		ReadOnly:  false,
		MountPath: "/etc/telegraf",
	}})
	ds, err := m.newDaemonSet(
		cType, oc, cfg, NewTelegrafVolume(cType, oc), dsSpec.DaemonSetSpec,
		"", initContainers, containersF,
	)
	ds.Spec.Template.Spec.ServiceAccountName = constants.ServiceAccountOnecloudOperator
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func NewTelegrafVolume(
	cType v1alpha1.ComponentType,
	oc *v1alpha1.OnecloudCluster,
) *VolumeHelper {
	var h = &VolumeHelper{
		cluster:      oc,
		component:    cType,
		volumes:      make([]corev1.Volume, 0),
		volumeMounts: make([]corev1.VolumeMount, 0),
	}
	h.volumeMounts = append(h.volumeMounts, []corev1.VolumeMount{
		{
			Name:      "etc-telegraf",
			ReadOnly:  false,
			MountPath: "/etc/telegraf",
		},
		{
			Name:      "proc",
			ReadOnly:  false,
			MountPath: "/proc",
		},
		{
			Name:      "sys",
			ReadOnly:  false,
			MountPath: "/sys",
		},
		{
			Name:      "log",
			ReadOnly:  false,
			MountPath: "/var/log/telegraf",
		},
		{
			Name:      "run",
			ReadOnly:  false,
			MountPath: "/var/run",
		},
	}...)

	var volSrcType = corev1.HostPathDirectoryOrCreate
	var hostPathDirectory = corev1.HostPathDirectory
	h.volumes = append(h.volumes, []corev1.Volume{
		{
			Name: "etc-telegraf",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/telegraf",
					Type: &volSrcType,
				},
			},
		},
		{
			Name: "proc",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/proc",
					Type: &hostPathDirectory,
				},
			},
		},
		{
			Name: "log",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log/telegraf",
					Type: &volSrcType,
				},
			},
		},
		{
			Name: "run",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run",
					Type: &hostPathDirectory,
				},
			},
		},
		{
			Name: "sys",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys",
					Type: &hostPathDirectory,
				},
			},
		},
	}...)
	return h
}
