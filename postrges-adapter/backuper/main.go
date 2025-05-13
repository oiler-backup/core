package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	_ "github.com/lib/pq"

	metricsbase "github.com/AntonShadrinNN/oiler-backup-base/metrics"
	s3base "github.com/AntonShadrinNN/oiler-backup-base/s3"
)

const (
	S3REGION = "us-east-1" // Fictious
)

func main() {
	ctx := context.Background()
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	coreAddr := os.Getenv("CORE_ADDR")
	backupPath := "/tmp/backup.sql"

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	backupName := fmt.Sprintf("%s:%s/%s", dbHost, dbPort, dbName)

	metricsReporter := metricsbase.NewMetricsReporter(coreAddr, false)
	s3Uploader, err := s3base.NewS3Uploader(ctx, s3Endpoint, s3AccessKey, s3SecretKey, S3REGION, false)
	if err != nil {
		log.Printf("Failed to initialize s3Uploder: %w", err)
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}
		os.Exit(1)
	}

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" ||
		s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3BucketName == "" || coreAddr == "" {
		log.Println("Envs DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET_NAME, CORE_ADDR are required")
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}
		os.Exit(1)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}

		os.Exit(1)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}
	}
	log.Println("Connection to database successful")

	dumpCmd := exec.CommandContext(ctx, "pg_dump",
		"-h", dbHost,
		"-p", dbPort,
		"-U", dbUser,
		"-d", dbName,
		"-F",
		"c",
		"-f", backupPath)
	dumpCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbPassword))

	output, err := dumpCmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed executing pg_dump: %v\n%s\n", err, string(output))
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}
	}
	log.Printf("Backup created successfully: %s\n", backupPath)

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	backupFile, err := os.Open(backupPath)
	if err != nil {
		log.Printf("Failed to open backupFile: %w", err)
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatalf("Failed to report metric %w\n", err)
		}
		os.Exit(1)
	}
	defer backupFile.Close()
	err = s3Uploader.Upload(ctx, s3BucketName, fmt.Sprintf("%s-%s-backup.sql", dbName, dateNow), backupFile)
	if err != nil {
		log.Printf("Failed to upload backup to MinIO: %v\n", err)
		err := metricsReporter.ReportStatus(ctx, backupName, false, -1)
		if err != nil {
			log.Fatal("Failed to report metric %w\n", err)
		}
	}

	err = metricsReporter.ReportStatus(ctx, backupName, true, time.Now().Unix())
	if err != nil {
		log.Fatalf("Failed to report metric %w\n", err)
	}
	log.Println("Backup successfully loaded to MinIO")
}
