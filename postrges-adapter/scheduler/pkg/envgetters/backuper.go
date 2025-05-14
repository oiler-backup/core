package envgetters

import (
	corev1 "k8s.io/api/core/v1"
)

type BackuperEnvGetter struct {
}

func (beg BackuperEnvGetter) GetEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{}
}
