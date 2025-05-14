package envgetters

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type BackuperEnvGetter struct {
	MaxBackupCount int
}

func (beg BackuperEnvGetter) GetEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "MAX_BACKUP_COUNT",
			Value: fmt.Sprint(beg.MaxBackupCount),
		},
	}
}
