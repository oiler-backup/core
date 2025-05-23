package main

import (
	"context"
	"fmt"
	"os"
	"restorer/internal/config"
	"restorer/internal/restorer"
	"time"

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

	restorer := restorer.NewRestorer(cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPassword, cfg.DbName, BACKUP_PATH)
	downloader, err := s3base.NewS3Downloader(ctx, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, S3REGION, cfg.Secure)
	if err != nil {
		mustProccessErrors("Failed to create downloader", err)
	}
	metricsReporter = metricsbase.NewMetricsReporter(cfg.CoreAddr, false)

	err = downloader.Download(ctx, cfg.S3BucketName, cfg.BackupRevision, BACKUP_PATH)
	if err != nil {
		mustProccessErrors("Failed to perform download", err)
	}

	err = restorer.Restore(ctx)
	if err != nil {
		mustProccessErrors("Faild to restore backup", err)
	}

	err = metricsReporter.ReportStatus(ctx, backupName, true, time.Now().Unix())
	if err != nil {
		mustProccessErrors("Failed to report successful status", err)
	}
	logger.Infof("Backup was applied successfully")
}

func mustProccessErrors(msg string, err error, keysAndValues ...any) {
	logger.Errorw(msg, "error", err, keysAndValues)
	err = metricsReporter.ReportStatus(ctx, backupName, false, -1)
	if err != nil {
		logger.Fatalf("Failed to report metric %w\n", err)
	}
	os.Exit(1)
}
