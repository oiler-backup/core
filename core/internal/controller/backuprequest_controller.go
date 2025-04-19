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

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/AntonShadrinNN/oiler-backup/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrNotSupported  = func(name string) error { return fmt.Errorf("Database %s is not supported", name) }
	ErrAlreadyExists = fmt.Errorf("Job already exists")
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
// +kubebuilder:rbac:groups="batch",resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

func (r *BackupRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cjExists := false
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

	if backupRequest.Status.Status == "In Progress" {
		log.Info("BackupRequest %s is already in progress, skipping...", "name", backupRequest.Name)
		return ctrl.Result{}, nil
	}

	backupRequest.Status.Status = "In Progress"
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		return ctrl.Result{}, err
	}

	controllerAddress, exists := r.DatabaseControllers[backupRequest.Spec.DatabaseType]
	if !exists {
		err := ErrNotSupported(backupRequest.Spec.DatabaseType)
		log.Error(err, "Make sure to update database-config cm")
		return ctrl.Result{}, err
	}

	cronJob, err := r.delegateToController(ctx, controllerAddress, &backupRequest)
	if errors.Is(err, ErrAlreadyExists) {
		log.Info("CronJob for BackupRequest %s already exists", "name", backupRequest.Name)
		cjExists = true
	} else if err != nil {
		log.Error(err, "Cannot delegate to controller")
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
			return ctrl.Result{}, err
		}
	}

	_, err = r.createCleanupJob(ctx, &backupRequest)
	if err != nil {
		log.Error(err, "Failed to create cleanupJob")
		return ctrl.Result{}, err
	}

	backupRequest.Status.Status = "Success"
	if err := r.Status().Update(ctx, &backupRequest); err != nil {
		log.Error(err, "Unable to update BackupRequest status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BackupRequestReconciler) delegateToController(ctx context.Context, controllerAddress string, backupRequest *backupv1.BackupRequest) (*batchv1.CronJob, error) {
	conn, err := grpc.Dial(controllerAddress, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %s: %w", controllerAddress, err)
	}
	defer conn.Close()

	client := NewBackupServiceClient(conn)

	req := &BackupRequest{
		DbUri:        backupRequest.Spec.DatabaseURI,
		DbPort:       int64(backupRequest.Spec.DatabasePort),
		DbUser:       backupRequest.Spec.DatabaseUser,
		DbPass:       backupRequest.Spec.DatabasePass,
		DbName:       backupRequest.Spec.DatabaseName,
		DatabaseType: backupRequest.Spec.DatabaseType,
		Schedule:     backupRequest.Spec.Schedule,
		StorageClass: backupRequest.Spec.StorageClass,
		S3Endpoint:   backupRequest.Spec.S3Endpoint,
		S3AccessKey:  backupRequest.Spec.S3AccessKey,
		S3SecretKey:  backupRequest.Spec.S3SecretKey,
		S3BucketName: backupRequest.Spec.S3BucketName,
	}

	resp, err := client.Backup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Failed to invoke backup method %s: %w", controllerAddress, err)
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

func (r *BackupRequestReconciler) createCleanupJob(ctx context.Context, req *backupv1.BackupRequest) (*batchv1.CronJob, error) {
	cleanerCronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cleaner-%s", req.Spec.DatabaseName),
			Namespace: "oiler-backup-system",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         req.APIVersion,
					Kind:               req.Kind,
					Name:               req.Name,
					UID:                req.UID,
					BlockOwnerDeletion: func() *bool { b := true; return &b }(),
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: req.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "cleaner-job",
									Image:           "ashadrinnn/cleaner:0.0.1-0",
									ImagePullPolicy: corev1.PullAlways,
									Env: []corev1.EnvVar{
										{
											Name:  "S3_ENDPOINT",
											Value: req.Spec.S3Endpoint,
										},
										{
											Name:  "S3_ACCESS_KEY",
											Value: req.Spec.S3AccessKey,
										},
										{
											Name:  "S3_SECRET_KEY",
											Value: req.Spec.S3SecretKey,
										},
										{
											Name:  "S3_BUCKET_NAME",
											Value: req.Spec.S3BucketName,
										},
										{
											Name:  "S3_BACKUP_DIR",
											Value: req.Spec.DatabaseName,
										},
										{
											Name:  "MAX_BACKUP_COUNT",
											Value: fmt.Sprint(req.Spec.MaxBackupCount),
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
	name := types.NamespacedName{
		Namespace: cleanerCronJob.Namespace,
		Name:      cleanerCronJob.Name,
	}
	var cj batchv1.CronJob
	err := r.Get(ctx, name, &cj)
	if apierrors.IsAlreadyExists(err) {
		return &cj, err
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	err = r.Create(ctx, cleanerCronJob)
	if err != nil {
		return nil, err
	}

	return cleanerCronJob, nil
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
