//go:build !test

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"backuper/internal/backuper"
	"backuper/internal/config"

	_ "github.com/lib/pq"
	loggerbase "github.com/oiler-backup/base/logger"
	metricsbase "github.com/oiler-backup/base/metrics"
	s3base "github.com/oiler-backup/base/s3"
	"go.uber.org/zap"
)

const (
	S3REGION    = "us-east-1" // Fictious
	BACKUP_PATH = "/tmp/backup.sql"
)

var (
	logger          *zap.SugaredLogger
	metricsReporter metricsbase.MetricsReporter
	ctx             context.Context
	backupName      string
)

func main() {
	ctx = context.Background()

	// Zap logger configuration
	var err error
	logger, err = loggerbase.GetLogger(loggerbase.PRODUCTION)
	if err != nil {
		panic(fmt.Sprintf("Failed to initiate logger: %v", err))
	}

	// Configuration of a backuper
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to configurate: %v", err))
	}
	backupName = fmt.Sprintf("%s:%s/%s", cfg.DbHost, cfg.DbPort, cfg.DbName)
	backuper := backuper.NewBackuper(cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPassword, cfg.DbName, BACKUP_PATH)
	s3UploaderCleaner, err := s3base.NewS3UploadCleaner(ctx, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, S3REGION, cfg.Secure)
	if err != nil {
		mustProccessErrors("Failed to initialize s3Uploader: %+v", err)
	}

	// Backward metrics reporter
	metricsReporter = metricsbase.NewMetricsReporter(cfg.CoreAddr, false)

	start := time.Now()
	err = backuper.Backup(ctx, cfg.Secure)
	if err != nil {
		mustProccessErrors("Failed to perform backup", err)
	}

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	backupFile, err := os.Open(BACKUP_PATH)
	if err != nil {
		mustProccessErrors("Failed to open backupFile: %+v", err)
	}
	defer backupFile.Close()
	err = s3UploaderCleaner.CleanAndUpload(ctx, cfg.S3BucketName, cfg.DbName, cfg.MaxBackupCount, fmt.Sprintf("%s/%s-backup.sql", cfg.DbName, dateNow), backupFile)
	if err != nil {
		mustProccessErrors("Failed to upload backup to S3: %+v", err)
	}

	timeElapsed := time.Since(start)
	err = metricsReporter.ReportStatus(ctx, backupName, true, int64(timeElapsed.Milliseconds()))
	if err != nil {
		logger.Fatalf("Failed to report successful status %w\n", err)
	}
	logger.Infof("Backup successfully loaded to S3")
}

func mustProccessErrors(msg string, err error, keysAndValues ...any) {
	logger.Errorw(msg, "error", err, keysAndValues)
	err = metricsReporter.ReportStatus(ctx, backupName, false, -1)
	if err != nil {
		logger.Fatalf("Failed to report metric %w\n", err)
	}
	os.Exit(1)
}
