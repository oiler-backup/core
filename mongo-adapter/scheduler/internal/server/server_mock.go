package server

import (
	"context"

	"github.com/oiler-backup/base/servers/backup/envgetters"
	"github.com/stretchr/testify/mock"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type MockJobsStub struct {
	mock.Mock
}

func (m *MockJobsStub) BuildBackuperCj(schedule string, eg envgetters.EnvGetter) *batchv1.CronJob {
	args := m.Called(schedule, eg)
	return args.Get(0).(*batchv1.CronJob)
}

func (m *MockJobsStub) BuildRestorerJob(envGetter envgetters.EnvGetter) *batchv1.Job {
	args := m.Called(envGetter)
	return args.Get(0).(*batchv1.Job)
}

type MockJobsCreator struct {
	mock.Mock
}

func (m *MockJobsCreator) CreateCronJob(ctx context.Context, cj *batchv1.CronJob) (string, string, error) {
	args := m.Called(ctx, cj)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockJobsCreator) UpdateCronJob(ctx context.Context, name, namespace string, envs []corev1.EnvVar) error {
	args := m.Called(ctx, name, namespace, envs)
	return args.Error(0)
}

func (m *MockJobsCreator) CreateJob(ctx context.Context, job *batchv1.Job) (string, string, error) {
	args := m.Called(ctx, job)
	return args.String(0), args.String(1), args.Error(2)
}
