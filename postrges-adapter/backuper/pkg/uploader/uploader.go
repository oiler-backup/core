package uploader

import (
	"context"
	"fmt"
	"os"
	"time"

	s3base "github.com/AntonShadrinNN/oiler-backup-base/s3"
)

type ErrUpload = error

func buildUploadError(msg string, opts ...any) ErrUpload {
	return fmt.Errorf(msg, opts)
}

type Uploader struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	dbName     string
	secure     bool

	region string
}

func NewUploader(endpoint, accessKey, secretKey, bucketName, dbName, region string, secure bool) Uploader {
	return Uploader{
		endpoint:   endpoint,
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucketName: bucketName,
		dbName:     dbName,
		secure:     secure,
		region:     region,
	}
}

func (u Uploader) Upload(ctx context.Context, filePath string) error {
	s3Uploader, err := s3base.NewS3Uploader(ctx, u.endpoint, u.accessKey, u.secretKey, u.region, u.secure)
	if err != nil {
		return buildUploadError("Failed to initialize s3Uploader: %+v", err)
	}

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	backupFile, err := os.Open(filePath)
	if err != nil {
		return buildUploadError("Failed to open backupFile: %+v", err)
	}
	defer backupFile.Close()
	err = s3Uploader.Upload(ctx, u.bucketName, fmt.Sprintf("%s-%s-backup.sql", u.dbName, dateNow), backupFile)
	if err != nil {
		return buildUploadError("Failed to upload backup to S3: %+v", err)
	}

	return nil
}
