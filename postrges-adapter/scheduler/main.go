package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	ErrAlreadyExists = fmt.Errorf("Already exists")
)

type BackupServer struct {
	UnimplementedBackupServiceServer
	kubeClient *kubernetes.Clientset
}

func NewBackupServer() (*BackupServer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &BackupServer{kubeClient: clientset}, nil
}

func (s *BackupServer) Backup(ctx context.Context, req *BackupRequest) (*BackupResponse, error) {
	log.Printf("Requested backup: DatabaseURI=%s, DatabaseType=%s, Schedule=%s, StorageClass=%s",
		req.DbUri, req.DatabaseType, req.Schedule, req.StorageClass)

	name, namespace, err := s.createCronJob(req)
	if errors.Is(err, ErrAlreadyExists) {
		return &BackupResponse{
			Status:           "Exists",
			CronjobName:      name,
			CronjobNamespace: namespace,
		}, nil
	}
	if err != nil {
		log.Printf("Failed to create CronJob: %v", err)
		return &BackupResponse{Status: "Failed to create CronJob"}, nil
	}

	return &BackupResponse{
		Status:           "CronJob created successfully",
		CronjobName:      name,
		CronjobNamespace: namespace,
	}, nil
}

func (s *BackupServer) createCronJob(req *BackupRequest) (string, string, error) {
	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	if podName == "" || namespace == "" {
		return "", "", fmt.Errorf("Failed to get POD_NAME and POD_NAMESPACE from envs")
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("backup-%s", req.DatabaseType),
			Namespace: "default",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: req.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "backup-job",
									Image:           "ashadrinnn/pgbackuper:0.0.1-0",
									ImagePullPolicy: corev1.PullAlways,
									Env: []corev1.EnvVar{
										{
											Name:  "DB_HOST",
											Value: req.DbUri,
										},
										{
											Name:  "DB_PORT",
											Value: fmt.Sprint(req.DbPort),
										},
										{
											Name:  "DB_USER",
											Value: req.DbUser,
										},
										{
											Name:  "DB_PASSWORD",
											Value: req.DbPass,
										},
										{
											Name:  "DB_NAME",
											Value: req.DbName,
										},
										{
											Name:  "S3_ENDPOINT",
											Value: req.S3Endpoint,
										},
										{
											Name:  "S3_ACCESS_KEY",
											Value: req.S3AccessKey,
										},
										{
											Name:  "S3_SECRET_KEY",
											Value: req.S3SecretKey,
										},
										{
											Name:  "S3_BUCKET_NAME",
											Value: req.S3BucketName,
										},
										{
											Name:  "BACKUP_PATH",
											Value: "./backup.sql",
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
	exCj, err := s.kubeClient.BatchV1().CronJobs(cronJob.Namespace).Get(context.TODO(), cronJob.Name, metav1.GetOptions{})
	if apierrors.IsAlreadyExists(err) {
		return exCj.Name, exCj.Namespace, ErrAlreadyExists
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return "", "", fmt.Errorf("Failed to check cj %s existence: %w", cronJob.Name, err)
	}
	generatedJob, err := s.kubeClient.BatchV1().CronJobs("default").Create(context.TODO(), cronJob, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create CronJob: %w", err)
	}

	log.Printf("CronJob '%s' created successfully", cronJob.Name)
	return generatedJob.Name, generatedJob.Namespace, nil
}

func (s *BackupServer) GetMetrics(ctx context.Context, req *MetricsRequest) (*MetricsResponse, error) {
	log.Printf("Got metrics response DatabaseType=%s", req.DatabaseType)

	return &MetricsResponse{
		SuccessfulBackups: 10,
		FailedBackups:     2,
	}, nil
}

func main() {
	server, err := NewBackupServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen port: %v", err)
	}

	grpcServer := grpc.NewServer()

	RegisterBackupServiceServer(grpcServer, server)

	log.Println("Running grpc server on port :50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed running server: %v", err)
	}
}
