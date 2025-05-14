package uploader

import (
	"context"
	"fmt"
	"os"
	"time"

	s3base "github.com/AntonShadrinNN/oiler-backup-base/s3"
)

type ErrUpload = error

func buildUploadCleanerError(msg string, opts ...any) ErrUpload {
	return fmt.Errorf(msg, opts)
}

type UploadCleaner struct {
	endpoint       string
	accessKey      string
	secretKey      string
	bucketName     string
	maxBackupCount int
	dbName         string
	secure         bool

	region string
}

func NewUploadCleaner(endpoint, accessKey, secretKey, bucketName, dbName, region string, maxBackupCount int, secure bool) UploadCleaner {
	return UploadCleaner{
		endpoint:       endpoint,
		accessKey:      accessKey,
		secretKey:      secretKey,
		bucketName:     bucketName,
		maxBackupCount: maxBackupCount,
		dbName:         dbName,
		secure:         secure,
		region:         region,
	}
}

func (uc UploadCleaner) Upload(ctx context.Context, filePath string) error {
	s3UploadeCleaner, err := s3base.NewS3UploadCleaner(ctx, uc.endpoint, uc.accessKey, uc.secretKey, uc.region, uc.secure)
	if err != nil {
		return buildUploadCleanerError("Failed to initialize s3Uploader: %+v", err)
	}

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	backupFile, err := os.Open(filePath)
	if err != nil {
		return buildUploadCleanerError("Failed to open backupFile: %+v", err)
	}
	defer backupFile.Close()
	err = s3UploadeCleaner.CleanAndUpload(ctx, uc.bucketName, uc.dbName, uc.maxBackupCount, fmt.Sprintf("%s/%s-backup.sql", uc.dbName, dateNow), backupFile)
	if err != nil {
		return buildUploadCleanerError("Failed to upload backup to S3: %+v", err)
	}

	return nil
}
