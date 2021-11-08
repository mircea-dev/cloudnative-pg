/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Package pgbouncer contains the specification of the K8s resources
// generated by the Cloud Native PostgreSQL operator related to pgbouncer poolers
package pgbouncer

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/EnterpriseDB/cloud-native-postgresql/api/v1"
	config "github.com/EnterpriseDB/cloud-native-postgresql/internal/configuration"
	pgBouncerConfig "github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/pgbouncer/config"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/url"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/podspec"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/postgres"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/specs"
)

const (
	// PgbouncerPoolerSpecHash is the annotation added to the deployment to tell
	// the hash of the Pooler Specification
	PgbouncerPoolerSpecHash = specs.MetadataNamespace + "/poolerSpecHash"

	// PgbouncerNameLabel is the label of the pgbouncer pod used by default
	PgbouncerNameLabel = specs.MetadataNamespace + "/poolerName"

	// DefaultPgbouncerImage is the name of the pgbouncer image used by default
	DefaultPgbouncerImage = "quay.io/enterprisedb/pgbouncer:1.16.0"
)

// Deployment create the deployment of pgbouncer, given
// the configurations we have in the pooler specifications
func Deployment(pooler *apiv1.Pooler,
	cluster *apiv1.Cluster,
) (*appsv1.Deployment, error) {
	podTemplate := podspec.NewFrom(pooler.Spec.Template).
		WithLabel(PgbouncerNameLabel, pooler.Name).
		WithVolume(&corev1.Volume{
			Name: "ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cluster.GetServerCASecretName(),
				},
			},
		}).
		WithVolume(&corev1.Volume{
			Name: "server-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cluster.GetServerTLSSecretName(),
				},
			},
		}).
		WithContainerImage("pgbouncer", DefaultPgbouncerImage, false).
		WithContainerCommand("pgbouncer", []string{
			"/controller/manager",
			"pgbouncer",
			"run",
		}, false).
		WithContainerPort("pgbouncer", &corev1.ContainerPort{
			Name:          "pgbouncer",
			ContainerPort: pgBouncerConfig.PgBouncerPort,
		}).
		WithContainerPort("pgbouncer", &corev1.ContainerPort{
			Name:          "metrics",
			ContainerPort: int32(url.PgBouncerMetricsPort),
		}).
		WithInitContainerImage(specs.BootstrapControllerContainerName, config.Current.OperatorImageName, true).
		WithInitContainerCommand(specs.BootstrapControllerContainerName,
			[]string{"/manager", "bootstrap", "/controller/manager"},
			true).
		WithVolume(&corev1.Volume{
			Name: "scratch-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}).
		WithInitContainerVolumeMount(specs.BootstrapControllerContainerName, &corev1.VolumeMount{
			Name:      "scratch-data",
			MountPath: postgres.ScratchDataDirectory,
		}, true).
		WithContainerVolumeMount("pgbouncer", &corev1.VolumeMount{
			Name:      "scratch-data",
			MountPath: postgres.ScratchDataDirectory,
		}, true).
		WithContainerEnv("pgbouncer", corev1.EnvVar{Name: "NAMESPACE", Value: pooler.Namespace}, true).
		WithContainerEnv("pgbouncer", corev1.EnvVar{Name: "POOLER_NAME", Value: pooler.Name}, true).
		WithServiceAccountName(pooler.Name, true).
		Build()

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pooler.Name,
			Namespace: pooler.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &pooler.Spec.Instances,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					PgbouncerNameLabel: pooler.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: podTemplate.ObjectMeta.Annotations,
					Labels:      podTemplate.ObjectMeta.Labels,
				},
				Spec: podTemplate.Spec,
			},
		},
	}, nil
}
