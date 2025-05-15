package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Status = string

const (
	SUCCESS     = "Success"
	FAILURE     = "Failure"
	IN_PROGRESS = "In Progress"
)

func loadDatabaseConfig(ctx context.Context, r client.Reader, namespace string) (map[string]string, error) {
	log := log.FromContext(ctx)
	configMap := &corev1.ConfigMap{}
	configMapName := "database-config"

	log.Info("Looking up for ConfigMap", "name", configMapName, "namespace", namespace)
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapName}, configMap); err != nil {
		return nil, fmt.Errorf("unable to get ConfigMap %s: %w", configMapName, err)
	}

	return configMap.Data, nil
}
