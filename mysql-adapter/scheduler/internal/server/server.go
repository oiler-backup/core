package server

import (
	"context"
	"errors"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	pb "github.com/oiler-backup/base/proto"
	serversbase "github.com/oiler-backup/base/servers/backup"
	eg "github.com/oiler-backup/base/servers/backup/envgetters"
)

// An ErrBackupServer is required for more verbosity.
type ErrBackupServer = error

// A BackupServer is an implementation of gRPC server to
// accept requests from Kubernetes Operator Core
// and create underlying resources.
type BackupServer struct {
	pb.UnimplementedBackupServiceServer
	kubeClient    *kubernetes.Clientset
	jobsCreator   serversbase.IJobsCreator
	namespace     string
	backuperImage string
	restorerImage string
	jobsStub      serversbase.IJobStub
}

// NewBackupServer is a constructor for BackupServer.
// Accepts systemNamespace where underlying resources will be created.
// backuperImg and restorerImg will be used as images in Kubernetes pods
func NewBackupServer(systemNamespace, backuperImg, restorerImg string) (*BackupServer, error) { // coverage-ignore
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
		"mysql",
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

func RegisterBackupServer(grpcServer *grpc.Server, systemNamespace, backuperImage, restorerImage string) error { // coverage-ignore
	server, err := NewBackupServer(systemNamespace, backuperImage, restorerImage)
	if err != nil {
		return err
	}
	pb.RegisterBackupServiceServer(grpcServer, server)

	return nil
}

// Backup creates CronJob with backuper image.
// Validates CronJob is actually created.
// Returns Status "Exists" in case of already created resource.
func (s *BackupServer) Backup(ctx context.Context, req *pb.BackupRequest) (*pb.BackupResponse, error) {
	cj := s.jobsStub.BuildBackuperCj(
		req.Schedule,
		eg.NewEnvGetterMerger([]eg.EnvGetter{
			eg.CommonEnvGetter{
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
			eg.BackuperEnvGetter{
				MaxBackupCount: int(req.MaxBackupCount),
			},
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

// Update performs update of a CronJob with backuper.
// Currently only changes environment variables.
func (s *BackupServer) Update(ctx context.Context, req *pb.UpdateBackupRequest) (*pb.BackupResponse, error) {
	err := s.jobsCreator.UpdateCronJob(
		ctx,
		req.CronjobName,
		req.CronjobNamespace,
		eg.NewEnvGetterMerger([]eg.EnvGetter{
			eg.CommonEnvGetter{
				DbUri:        req.Request.DbUri,
				DbPort:       fmt.Sprint(req.Request.DbPort),
				DbUser:       req.Request.DbUser,
				DbPass:       req.Request.DbPass,
				DbName:       req.Request.DbName,
				S3Endpoint:   req.Request.S3Endpoint,
				S3AccessKey:  req.Request.S3AccessKey,
				S3SecretKey:  req.Request.S3SecretKey,
				S3BucketName: req.Request.S3BucketName,
				CoreAddr:     req.Request.CoreAddr,
			},
			eg.BackuperEnvGetter{
				MaxBackupCount: int(req.Request.MaxBackupCount),
			},
		}).GetEnvs(),
	)
	if err != nil {
		return &pb.BackupResponse{
			Status: "Failed to update cronjob",
		}, err
	}

	return &pb.BackupResponse{
		Status:           "CronJob updated successfully",
		CronjobName:      req.CronjobName,
		CronjobNamespace: req.CronjobNamespace,
	}, nil
}

// Restore restores backup from s3-compatible storage.
func (s *BackupServer) Restore(ctx context.Context, req *pb.BackupRestore) (*pb.BackupRestoreResponse, error) {
	job := s.jobsStub.BuildRestorerJob(
		eg.NewEnvGetterMerger([]eg.EnvGetter{
			eg.CommonEnvGetter{
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
			eg.RestorerEnvGetter{
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
