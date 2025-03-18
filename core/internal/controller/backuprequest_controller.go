/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/AntonShadrinNN/oiler-backup/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// BackupRequestReconciler reconciles a BackupRequest object
type BackupRequestReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	DatabaseControllers map[string]string
}

// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
func (r *BackupRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("backuprequest", req.NamespacedName)
	if err := r.loadDatabaseConfig(context.Background(), "default"); err != nil {
		log.Error(err, "Failed to load config")
		return ctrl.Result{}, err
	}
	var backupRequest backupv1.BackupRequest
	if err := r.Get(ctx, req.NamespacedName, &backupRequest); err != nil {
		log.Error(err, "Unable to get BackupRequest object")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	controllerAddress, exists := r.DatabaseControllers[backupRequest.Spec.DatabaseType]
	if !exists {
		log.Info(fmt.Sprintf("Database type %s is not supported", backupRequest.Spec.DatabaseType))
		return ctrl.Result{}, nil
	}

	err := r.delegateToController(ctx, controllerAddress, &backupRequest)
	if err != nil {
		log.Error(err, "Cannot delegate to controller")
		return ctrl.Result{}, err
	}

	backupRequest.Status.Status = "In Progress"
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BackupRequestReconciler) delegateToController(ctx context.Context, controllerAddress string, backupRequest *backupv1.BackupRequest) error {
	conn, err := grpc.Dial(controllerAddress, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %w", controllerAddress, err)
	}
	defer conn.Close()

	client := NewBackupServiceClient(conn)

	req := &BackupRequest{
		DatabaseFqdn:    backupRequest.Spec.DatabaseURI,
		StorageLocation: backupRequest.Spec.StorageClass,
	}

	resp, err := client.Backup(ctx, req)
	if err != nil {
		return fmt.Errorf("Failed to invoke backup method %s: %w", controllerAddress, err)
	}

	log.FromContext(ctx).Info(resp.String())
	return nil
}

func (r *BackupRequestReconciler) loadDatabaseConfig(ctx context.Context, namespace string) error {
	configMap := &corev1.ConfigMap{}
	configMapName := "database-config"

	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapName}, configMap); err != nil {
		return fmt.Errorf("Unable to get ConfigMap %s: %w", configMapName, err)
	}

	r.DatabaseControllers = configMap.Data
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupRequest{}).
		Named("backuprequest").
		Complete(r)
}
