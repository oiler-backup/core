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
	"os"

	pb "github.com/oiler-backup/base/proto"
	backupv1 "github.com/oiler-backup/core/api/v1"
	config "github.com/oiler-backup/core/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	ErrNotSupported  = func(name string) error { return fmt.Errorf("database %s is not supported", name) }
	ErrAlreadyExists = fmt.Errorf("job already exists")
)

var (
	appCfg config.Config
)

// BackupRequestReconciler reconciles a BackupRequest object
type BackupRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.oiler.backup,resources=backuprequests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="batch",resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

func (r *BackupRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cjExists := false
	log := log.FromContext(ctx).WithValues("backuprequest", req.NamespacedName)
	var backupRequest backupv1.BackupRequest
	err := r.Get(ctx, req.NamespacedName, &backupRequest)
	if apierrors.IsNotFound(err) {
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

	controllerAddress, exists := dbControllers[backupRequest.Spec.DatabaseType]
	if !exists {
		err := ErrNotSupported(backupRequest.Spec.DatabaseType)
		log.Error(err, "Make sure to update database-config cm")
		innerErr := r.setFailed(ctx, req.NamespacedName)
		if innerErr != nil {
			return ctrl.Result{}, innerErr
		}
		return ctrl.Result{}, err
	}

	if backupRequest.Status.Status == SUCCESS {
		err := r.updateCronJob(ctx, controllerAddress, &backupRequest)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	backupRequest.Status.Status = IN_PROGRESS
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		innerErr := r.setFailed(ctx, req.NamespacedName)
		if innerErr != nil {
			return ctrl.Result{}, innerErr
		}
		return ctrl.Result{}, err
	}

	cronJob, err := r.delegateToController(ctx, controllerAddress, &backupRequest)
	if errors.Is(err, ErrAlreadyExists) {
		log.Info("CronJob for BackupRequest already exists", "name", backupRequest.Name)
		cjExists = true
	} else if err != nil {
		log.Error(err, "Cannot delegate to controller")
		innerErr := r.setFailed(ctx, req.NamespacedName)
		if innerErr != nil {
			return ctrl.Result{}, innerErr
		}
		return ctrl.Result{}, err
	}

	if !cjExists {
		cronJob.OwnerReferences = append(cronJob.OwnerReferences, metav1.OwnerReference{
			APIVersion:         backupRequest.APIVersion,
			Kind:               backupRequest.Kind,
			Name:               backupRequest.Name,
			UID:                backupRequest.UID,
			BlockOwnerDeletion: func() *bool { b := true; return &b }(),
		})

		if err := r.Update(ctx, cronJob); err != nil {
			log.Error(err, "Failed to update cronJob")
			innerErr := r.setFailed(ctx, req.NamespacedName)
			if innerErr != nil {
				return ctrl.Result{}, innerErr
			}
			return ctrl.Result{}, err
		}
	}

	backupRequest.Status.Status = SUCCESS
	backupRequest.Status.CronJobData = backupv1.CreatedCronJobData{
		Name:      cronJob.Name,
		Namespace: cronJob.Namespace,
	}
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		innerErr := r.setFailed(ctx, req.NamespacedName)
		if innerErr != nil {
			return ctrl.Result{}, innerErr
		}
		return ctrl.Result{}, err
	}

	log.Info("Successfully created all resources")
	return ctrl.Result{}, nil
}

func (r *BackupRequestReconciler) setFailed(ctx context.Context, nsName types.NamespacedName) error {
	log := log.FromContext(ctx).WithValues("set-failed", nsName)
	var backupRequest backupv1.BackupRequest
	if err := r.Get(ctx, nsName, &backupRequest); err != nil {
		return err
	}

	if backupRequest.Status.Status == SUCCESS {
		return nil
	}

	backupRequest.Status.Status = FAILURE
	log.Info("Setting BackupRequest failed")
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Failed to set failed status on br")
		return err
	}

	log.Info("Successfully updated BackupRequest status to failed state")
	return nil
}

func (r *BackupRequestReconciler) delegateToController(ctx context.Context, controllerAddress string, backupRequest *backupv1.BackupRequest) (*batchv1.CronJob, error) {
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

	req := &pb.BackupRequest{
		DbUri:          backupRequest.Spec.DatabaseURI,
		DbPort:         int64(backupRequest.Spec.DatabasePort),
		DbUser:         backupRequest.Spec.DatabaseUser,
		DbPass:         backupRequest.Spec.DatabasePass,
		DbName:         backupRequest.Spec.DatabaseName,
		DatabaseType:   backupRequest.Spec.DatabaseType,
		Schedule:       backupRequest.Spec.Schedule,
		StorageClass:   backupRequest.Spec.StorageClass,
		S3Endpoint:     backupRequest.Spec.S3Endpoint,
		S3AccessKey:    backupRequest.Spec.S3AccessKey,
		S3SecretKey:    backupRequest.Spec.S3SecretKey,
		S3BucketName:   backupRequest.Spec.S3BucketName,
		CoreAddr:       os.Getenv("CORE_ADDR"),
		MaxBackupCount: backupRequest.Spec.MaxBackupCount,
	}

	resp, err := client.Backup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke backup method %s: %w", controllerAddress, err)
	}
	if resp.Status == "Exists" {
		return nil, ErrAlreadyExists
	}

	log.FromContext(ctx).Info(resp.String())

	name := types.NamespacedName{
		Namespace: resp.CronjobNamespace,
		Name:      resp.CronjobName,
	}
	var cronJob batchv1.CronJob
	err = r.Get(ctx, name, &cronJob)
	if err != nil {
		return nil, err
	}

	return &cronJob, nil
}

func (r *BackupRequestReconciler) updateCronJob(ctx context.Context, controllerAddress string, backupRequest *backupv1.BackupRequest) error {
	conn, err := grpc.NewClient(
		controllerAddress,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", controllerAddress, err)
	}
	defer conn.Close()

	client := pb.NewBackupServiceClient(conn)

	br := &pb.BackupRequest{
		DbUri:          backupRequest.Spec.DatabaseURI,
		DbPort:         int64(backupRequest.Spec.DatabasePort),
		DbUser:         backupRequest.Spec.DatabaseUser,
		DbPass:         backupRequest.Spec.DatabasePass,
		DbName:         backupRequest.Spec.DatabaseName,
		DatabaseType:   backupRequest.Spec.DatabaseType,
		Schedule:       backupRequest.Spec.Schedule,
		StorageClass:   backupRequest.Spec.StorageClass,
		S3Endpoint:     backupRequest.Spec.S3Endpoint,
		S3AccessKey:    backupRequest.Spec.S3AccessKey,
		S3SecretKey:    backupRequest.Spec.S3SecretKey,
		S3BucketName:   backupRequest.Spec.S3BucketName,
		CoreAddr:       os.Getenv("CORE_ADDR"),
		MaxBackupCount: backupRequest.Spec.MaxBackupCount,
	}

	req := pb.UpdateBackupRequest{
		Request:          br,
		CronjobName:      backupRequest.Status.CronJobData.Name,
		CronjobNamespace: backupRequest.Status.CronJobData.Namespace,
	}

	_, err = client.Update(ctx, &req)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	appCfg, err = config.GetConfig()
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupRequest{}).
		Named("backuprequest").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
