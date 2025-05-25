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

// Constants for S3 region and backup path.
const (
	S3REGION    = "us-east-1" // Fictitious
	BACKUP_PATH = "/tmp/backup.sql"
)

// Global variables for logger, metrics reporter, context, and backup name.
var (
	logger          *zap.SugaredLogger
	metricsReporter metricsbase.MetricsReporter
	ctx             context.Context
	backupName      string
)

// main initializes the logger, configuration, restorer, metrics reporter,
// and S3 downloader. It then downloads the backup file from S3, restores it,
// reports the status, and logs the success message.
func main() {
	ctx = context.Background()

	// Initialize the Zap logger with production settings.
	var err error
	logger, err = loggerbase.GetLogger(loggerbase.PRODUCTION)
	if err != nil {
		panic(fmt.Sprintf("Failed to initiate logger: %v", err))
	}

	// Load configuration settings.
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to configurate: %v", err))
	}

	// Create a new MetricsReporter instance with the provided configuration.
	metricsReporter = metricsbase.NewMetricsReporter(cfg.CoreAddr, false)
	// Create a new Restorer instance with the provided configuration.
	restorer := restorer.NewRestorer(cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPassword, cfg.DbName, BACKUP_PATH)
	// Create a new S3Downloader instance with the provided configuration.
	downloader, err := s3base.NewS3Downloader(ctx, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, S3REGION, cfg.Secure)
	if err != nil {
		mustProccessErrors("Failed to create downloader", err)
	}

	// Open a backup file for writing.
	backupFile, err := os.OpenFile(BACKUP_PATH, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		mustProccessErrors("Failed to open backupFile: %+v", err)
	}

	// Download the backup file from S3.
	err = downloader.Download(ctx, cfg.S3BucketName, cfg.DbName, cfg.BackupRevision, backupFile)
	if err != nil {
		mustProccessErrors("Failed to perform download", err)
	}

	// Restore the backup file to the MongoDB database.
	err = restorer.Restore(ctx)
	if err != nil {
		mustProccessErrors("Faild to restore backup", err)
	}

	// Report the successful restoration status.
	err = metricsReporter.ReportStatus(ctx, backupName, true, time.Now().Unix())
	if err != nil {
		mustProccessErrors("Failed to report successful status", err)
	}
	// Log the success message.
	logger.Infof("Backup was applied successfully")
}

// mustProccessErrors logs an error message and attempts to report the failure status.
// If reporting the failure status also fails, it logs a fatal error and exits the program.
func mustProccessErrors(msg string, err error, keysAndValues ...any) {
	logger.Errorw(msg, "error", err, keysAndValues)
	err = metricsReporter.ReportStatus(ctx, backupName, false, -1)
	if err != nil {
		logger.Fatalf("Failed to report metric %w\n", err)
	}
	os.Exit(1)
}
