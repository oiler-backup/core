package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pb "github.com/AntonShadrinNN/oiler-backup-base/proto"
	serversbase "github.com/AntonShadrinNN/oiler-backup-base/servers/backup"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func Test_Backup(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRequest{
		Schedule:       "0 0 * * *",
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		CoreAddr:       "http://core:8080",
		MaxBackupCount: 5,
	}

	cj := &batchv1.CronJob{}

	mockJobsStub.On("BuildBackuperCj", req.Schedule, mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(cj)

	mockJobsCreator.On("CreateCronJob", mock.Anything, cj).Return("cj-name", "default", nil)

	resp, err := server.Backup(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "CronJob created successfully", resp.Status)
	assert.Equal(t, "cj-name", resp.CronjobName)
	assert.Equal(t, "default", resp.CronjobNamespace)

	mockJobsStub.AssertExpectations(t)
	mockJobsCreator.AssertExpectations(t)
}

func Test_Backup_AlreadyExists(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRequest{
		Schedule:       "0 0 * * *",
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		CoreAddr:       "http://core:8080",
		MaxBackupCount: 5,
	}

	cj := &batchv1.CronJob{}

	mockJobsStub.On("BuildBackuperCj", req.Schedule, mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(cj)
	mockJobsCreator.On("CreateCronJob", mock.Anything, cj).Return("cj-name", "default", serversbase.ErrAlreadyExists)

	resp, err := server.Backup(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Exists", resp.Status)
	assert.Equal(t, "cj-name", resp.CronjobName)
	assert.Equal(t, "default", resp.CronjobNamespace)
}

func Test_Backup_CJ_CreationError(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRequest{
		Schedule:       "0 0 * * *",
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		CoreAddr:       "http://core:8080",
		MaxBackupCount: 5,
	}

	cj := &batchv1.CronJob{}

	mockJobsStub.On("BuildBackuperCj", req.Schedule, mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(cj)
	mockJobsCreator.On("CreateCronJob", mock.Anything, cj).Return("cj-name", "default", fmt.Errorf("some error"))

	resp, err := server.Backup(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, "Failed to create CronJob", resp.Status)
	assert.Empty(t, resp.CronjobName)
	assert.Empty(t, resp.CronjobNamespace)
}

func Test_Update(t *testing.T) {
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.UpdateBackupRequest{
		CronjobName:      "old-cj",
		CronjobNamespace: "default",
		Request: &pb.BackupRequest{
			DbUri:          "localhost",
			DbPort:         5432,
			DbUser:         "user",
			DbPass:         "pass",
			DbName:         "mydb",
			S3Endpoint:     "s3.example.com",
			S3AccessKey:    "key",
			S3SecretKey:    "secret",
			S3BucketName:   "bucket",
			CoreAddr:       "http://core:8080",
			MaxBackupCount: 5,
		},
	}

	expectedEnvs := []corev1.EnvVar{
		{Name: "DB_HOST", Value: req.Request.DbUri},
		{Name: "DB_PORT", Value: fmt.Sprint(req.Request.DbPort)},
		{Name: "DB_USER", Value: req.Request.DbUser},
		{Name: "DB_PASSWORD", Value: req.Request.DbPass},
		{Name: "DB_NAME", Value: req.Request.DbName},
		{Name: "S3_ENDPOINT", Value: req.Request.S3Endpoint},
		{Name: "S3_ACCESS_KEY", Value: req.Request.S3AccessKey},
		{Name: "S3_SECRET_KEY", Value: req.Request.S3SecretKey},
		{Name: "S3_BUCKET_NAME", Value: req.Request.S3BucketName},
		{Name: "CORE_ADDR", Value: req.Request.CoreAddr},
		{Name: "MAX_BACKUP_COUNT", Value: fmt.Sprint(req.Request.MaxBackupCount)},
	}

	mockJobsCreator.On("UpdateCronJob", mock.Anything, req.CronjobName, req.CronjobNamespace, expectedEnvs).Return(nil)

	resp, err := server.Update(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "CronJob updated successfully", resp.Status)
	assert.Equal(t, "old-cj", resp.CronjobName)
	assert.Equal(t, "default", resp.CronjobNamespace)

	mockJobsCreator.AssertExpectations(t)
}

func Test_Update_Error(t *testing.T) {
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.UpdateBackupRequest{
		CronjobName:      "old-cj",
		CronjobNamespace: "default",
		Request: &pb.BackupRequest{
			DbUri:          "localhost",
			DbPort:         5432,
			DbUser:         "user",
			DbPass:         "pass",
			DbName:         "mydb",
			S3Endpoint:     "s3.example.com",
			S3AccessKey:    "key",
			S3SecretKey:    "secret",
			S3BucketName:   "bucket",
			CoreAddr:       "http://core:8080",
			MaxBackupCount: 5,
		},
	}

	expectedEnvs := []corev1.EnvVar{
		{Name: "DB_HOST", Value: req.Request.DbUri},
		{Name: "DB_PORT", Value: fmt.Sprint(req.Request.DbPort)},
		{Name: "DB_USER", Value: req.Request.DbUser},
		{Name: "DB_PASSWORD", Value: req.Request.DbPass},
		{Name: "DB_NAME", Value: req.Request.DbName},
		{Name: "S3_ENDPOINT", Value: req.Request.S3Endpoint},
		{Name: "S3_ACCESS_KEY", Value: req.Request.S3AccessKey},
		{Name: "S3_SECRET_KEY", Value: req.Request.S3SecretKey},
		{Name: "S3_BUCKET_NAME", Value: req.Request.S3BucketName},
		{Name: "CORE_ADDR", Value: req.Request.CoreAddr},
		{Name: "MAX_BACKUP_COUNT", Value: fmt.Sprint(req.Request.MaxBackupCount)},
	}

	mockJobsCreator.On("UpdateCronJob", mock.Anything, req.CronjobName, req.CronjobNamespace, expectedEnvs).Return(fmt.Errorf("some error"))

	resp, err := server.Update(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, resp.Status, "Failed to update cronjob")
	assert.Empty(t, resp.CronjobName)
	assert.Empty(t, resp.CronjobNamespace)

	mockJobsCreator.AssertExpectations(t)
}

func Test_Restore(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRestore{
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		BackupRevision: "revision",
	}

	job := &batchv1.Job{}
	mockJobsStub.On("BuildRestorerJob", mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(job)
	mockJobsCreator.On("CreateJob", mock.Anything, job).Return("job-name", "default", nil)

	resp, err := server.Restore(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Job created successfully", resp.Status)
	assert.Equal(t, "job-name", resp.JobName)
	assert.Equal(t, "default", resp.JobNamespace)
}

func Test_Restore_AlreadyExists(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRestore{
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		BackupRevision: "revision",
	}

	job := &batchv1.Job{}
	mockJobsStub.On("BuildRestorerJob", mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(job)
	mockJobsCreator.On("CreateJob", mock.Anything, job).Return("job-name", "default", serversbase.ErrAlreadyExists)

	resp, err := server.Restore(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Exists", resp.Status)
	assert.Equal(t, "job-name", resp.JobName)
	assert.Equal(t, "default", resp.JobNamespace)
}

func Test_Restore_Job_CreationError(t *testing.T) {
	mockJobsStub := new(MockJobsStub)
	mockJobsCreator := new(MockJobsCreator)

	server := &BackupServer{
		jobsStub:    mockJobsStub,
		jobsCreator: mockJobsCreator,
		namespace:   "default",
	}

	req := &pb.BackupRestore{
		DbUri:          "localhost",
		DbPort:         5432,
		DbUser:         "user",
		DbPass:         "pass",
		DbName:         "mydb",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "key",
		S3SecretKey:    "secret",
		S3BucketName:   "bucket",
		BackupRevision: "revision",
	}

	job := &batchv1.Job{}
	mockJobsStub.On("BuildRestorerJob", mock.AnythingOfType("envgetters.EnvGetterMerger")).Return(job)
	mockJobsCreator.On("CreateJob", mock.Anything, job).Return("job-name", "default", fmt.Errorf("some error"))

	resp, err := server.Restore(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, "Failed to create Job", resp.Status)
	assert.Empty(t, resp.JobName)
	assert.Empty(t, resp.JobNamespace)
}
