package envgetters

import (
	corev1 "k8s.io/api/core/v1"
)

type RestorerEnvGetter struct {
	BackupRevision string
}

func (reg RestorerEnvGetter) GetEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "BACKUP_REVISION",
			Value: reg.BackupRevision,
		},
	}
}
