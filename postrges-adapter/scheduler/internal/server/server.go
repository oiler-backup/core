package server

import (
	"context"
	"errors"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"postgres_adapter/pkg/envgetters"

	pb "github.com/AntonShadrinNN/oiler-backup-base/proto"
	serversbase "github.com/AntonShadrinNN/oiler-backup-base/servers/backup"
)

type ErrBackupServer = error

type BackupServer struct {
	pb.UnimplementedBackupServiceServer
	kubeClient    *kubernetes.Clientset
	jobsCreator   serversbase.JobsCreator
	namespace     string
	backuperImage string
	restorerImage string
	coreAddr      string
	jobsStub      serversbase.JobsStub
}

func NewBackupServer(systemNamespace, backuperImg, restorerImg string) (*BackupServer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	jobsCreator := serversbase.NewJobsCreator(clientset)
	jobsStub := serversbase.NewJobsStub(
		"postgres",
		systemNamespace,
		backuperImg,
		restorerImg,
	)
	return &BackupServer{
		kubeClient:    clientset,
		jobsCreator:   jobsCreator,
		namespace:     systemNamespace,
		backuperImage: backuperImg,
		restorerImage: restorerImg,
		jobsStub:      jobsStub,
	}, nil
}

func RegisterBackupServer(grpcServer *grpc.Server, systemNamespace, backuperImage, restorerImage string) error {
	server, err := NewBackupServer(systemNamespace, backuperImage, restorerImage)
	if err != nil {
		return err
	}
	pb.RegisterBackupServiceServer(grpcServer, server)

	return nil
}

func (s *BackupServer) Backup(ctx context.Context, req *pb.BackupRequest) (*pb.BackupResponse, error) {
	cj := s.jobsStub.BuildBackuperCj(
		req.Schedule,
		serversbase.NewEnvGetterMerger([]serversbase.EnvGetter{
			envgetters.CommonEnvGetter{
				DbUri:        req.DbUri,
				DbPort:       fmt.Sprint(req.DbPort),
				DbUser:       req.DbUser,
				DbPass:       req.DbPass,
				DbName:       req.DbName,
				S3Endpoint:   req.S3Endpoint,
				S3AccessKey:  req.S3AccessKey,
				S3SecretKey:  req.S3SecretKey,
				S3BucketName: req.S3BucketName,
				CoreAddr:     req.CoreAddr,
			},
			envgetters.BackuperEnvGetter{},
		}),
	)
	name, namespace, err := s.jobsCreator.CreateCronJob(ctx, cj)
	if errors.Is(err, serversbase.ErrAlreadyExists) {
		return &pb.BackupResponse{
			Status:           "Exists",
			CronjobName:      name,
			CronjobNamespace: namespace,
		}, nil
	}
	if err != nil {
		log.Printf("Failed to create CronJob: %v", err)
		return &pb.BackupResponse{Status: "Failed to create CronJob"}, nil
	}

	return &pb.BackupResponse{
		Status:           "CronJob created successfully",
		CronjobName:      name,
		CronjobNamespace: namespace,
	}, nil
}

func (s *BackupServer) Restore(ctx context.Context, req *pb.BackupRestore) (*pb.BackupRestoreResponse, error) {
	job := s.jobsStub.BuildRestorerJob(
		serversbase.NewEnvGetterMerger([]serversbase.EnvGetter{
			envgetters.CommonEnvGetter{
				DbUri:        req.DbUri,
				DbPort:       fmt.Sprint(req.DbPort),
				DbUser:       req.DbUser,
				DbPass:       req.DbPass,
				DbName:       req.DbName,
				S3Endpoint:   req.S3Endpoint,
				S3AccessKey:  req.S3AccessKey,
				S3SecretKey:  req.S3SecretKey,
				S3BucketName: req.S3BucketName,
				// CoreAddr:     req.CoreAddr, // TODO
			},
			envgetters.RestorerEnvGetter{
				BackupRevision: req.BackupRevision,
			},
		},
		),
	)
	name, namespace, err := s.jobsCreator.CreateJob(ctx, job)
	if errors.Is(err, serversbase.ErrAlreadyExists) {
		return &pb.BackupRestoreResponse{
			Status:       "Exists",
			JobName:      name,
			JobNamespace: namespace,
		}, nil
	}
	if err != nil {
		log.Printf("Failed to create Job: %v", err)
		return &pb.BackupRestoreResponse{Status: "Failed to create Job"}, nil
	}

	return &pb.BackupRestoreResponse{
		Status:       "Job created successfully",
		JobName:      name,
		JobNamespace: namespace,
	}, nil
}
