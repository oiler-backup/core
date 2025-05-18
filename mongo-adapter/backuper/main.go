//go:build !test

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"backuper/internal/backuper"
	"backuper/internal/config"
	"backuper/pkg/uploader"

	loggerbase "github.com/oiler-backup/base/logger"
	metricsbase "github.com/oiler-backup/base/metrics"
	"go.uber.org/zap"
)

const (
	S3REGION    = "us-east-1" // Fictious
	BACKUP_PATH = "/tmp/backup.tar"
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
	uploader := uploader.NewUploadCleaner(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3BucketName, cfg.DbName, S3REGION, cfg.MaxBackupCount, cfg.Secure)

	// Backward metrics reporter
	metricsReporter = metricsbase.NewMetricsReporter(cfg.CoreAddr, false)

	err = backuper.Backup(ctx)
	if err != nil {
		mustProccessErrors("Failed to perform backup", err)
	}

	err = uploader.Upload(ctx, BACKUP_PATH)
	if err != nil {
		mustProccessErrors("Failed to perform upload", err)
	}

	err = metricsReporter.ReportStatus(ctx, backupName, true, time.Now().Unix())
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
