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
	"errors"
	"fmt"

	pb "github.com/AntonShadrinNN/oiler-backup-base/proto"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/AntonShadrinNN/oiler-backup/api/v1"
)

// BackupRestoreReconciler reconciles a BackupRestore object
type BackupRestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprestores/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *BackupRestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	jExists := false
	log := log.FromContext(ctx).WithValues("backuprestore", req.NamespacedName)
	var backupRestore backupv1.BackupRestore

	err := r.Get(ctx, req.NamespacedName, &backupRestore)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if backupRestore.Status.Status != "" {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Unable to get BackupRequest object")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	dbControllers, err := loadDatabaseConfig(ctx, r, appCfg.OperatorNamespace)
	if err != nil {
		log.Error(err, "Failed to load config")
		return ctrl.Result{}, err
	}

	backupRestore.Status.Status = IN_PROGRESS
	if err := r.Status().Update(ctx, &backupRestore); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		r.mustSetFailed(ctx, req.NamespacedName)
		return ctrl.Result{}, err
	}

	controllerAddress, exists := dbControllers[backupRestore.Spec.DatabaseType]
	if !exists {
		err := ErrNotSupported(backupRestore.Spec.DatabaseType)
		log.Error(err, "Make sure to update database-config cm")
		return ctrl.Result{}, err
	}

	job, err := r.delegateToController(ctx, controllerAddress, &backupRestore)
	if errors.Is(err, ErrAlreadyExists) {
		log.Info("Job for BackupRestore %s already exists", "name", backupRestore.Name)
		jExists = true
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Cannot delegate to controller")
		return ctrl.Result{}, err
	}

	if !jExists {
		job.OwnerReferences = append(job.OwnerReferences, metav1.OwnerReference{
			APIVersion:         backupRestore.APIVersion,
			Kind:               backupRestore.Kind,
			Name:               backupRestore.Name,
			UID:                backupRestore.UID,
			BlockOwnerDeletion: func() *bool { b := true; return &b }(),
		})

		if err := r.Update(ctx, job); err != nil {
			log.Error(err, "Failed to update job")
			r.mustSetFailed(ctx, req.NamespacedName)
			return ctrl.Result{}, err
		}
	}

	backupRestore.Status.Status = SUCCESS
	if err := r.Status().Update(ctx, &backupRestore); err != nil {
		log.Error(err, "Unable to update BackupRestore status")
		r.mustSetFailed(ctx, req.NamespacedName)
		return ctrl.Result{}, err
	}

	log.Info("Successfully created all resources")
	return ctrl.Result{}, nil
}

func (r *BackupRestoreReconciler) mustSetFailed(ctx context.Context, nsName types.NamespacedName) {
	log := log.FromContext(ctx).WithValues("set-failed", nsName)
	var backupRestore backupv1.BackupRestore
	if err := r.Get(ctx, nsName, &backupRestore); err != nil {
		panic(err)
	}

	if backupRestore.Status.Status == FAILURE || backupRestore.Status.Status == SUCCESS {
		return
	}

	backupRestore.Status.Status = FAILURE
	log.Info("Setting BackupRestore failed")
	if err := r.Status().Update(ctx, &backupRestore); err != nil {
		log.Error(err, "Failed to set failed status on br")
		panic(err)
	}

	log.Info("Successfully updated BackupRequest status to failed state")
}

func (r *BackupRestoreReconciler) delegateToController(ctx context.Context, controllerAddress string, backupRestore *backupv1.BackupRestore) (*batchv1.Job, error) {
	conn, err := grpc.NewClient(
		controllerAddress,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", controllerAddress, err)
	}
	defer conn.Close()

	client := pb.NewBackupServiceClient(conn)

	req := &pb.BackupRestore{
		DbUri:          backupRestore.Spec.DatabaseURI,
		DbPort:         int64(backupRestore.Spec.DatabasePort),
		DbUser:         backupRestore.Spec.DatabaseUser,
		DbPass:         backupRestore.Spec.DatabasePass,
		DbName:         backupRestore.Spec.DatabaseName,
		DatabaseType:   backupRestore.Spec.DatabaseType,
		S3Endpoint:     backupRestore.Spec.S3Endpoint,
		S3AccessKey:    backupRestore.Spec.S3AccessKey,
		S3SecretKey:    backupRestore.Spec.S3SecretKey,
		S3BucketName:   backupRestore.Spec.S3BucketName,
		BackupRevision: backupRestore.Spec.BackupRevision,
	}

	resp, err := client.Restore(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke restore method %s: %w", controllerAddress, err)
	}
	if resp.Status == "Exists" {
		return nil, ErrAlreadyExists
	}

	log.FromContext(ctx).Info(resp.String())

	name := types.NamespacedName{
		Namespace: resp.JobNamespace,
		Name:      resp.JobName,
	}
	var job batchv1.Job
	err = r.Get(ctx, name, &job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupRestore{}).
		Named("backuprestore").
		Complete(r)
}
