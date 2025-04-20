package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	ctx := context.Background()
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	backupPath := "/tmp/backup.sql"

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3BucketName := os.Getenv("S3_BUCKET_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" ||
		s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3BucketName == "" {
		log.Fatal("Envs DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET_NAME are required")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
		log.Fatalf("Failed executing pg_dump: %v\n%s", err, string(output))
	}
	log.Printf("Backup created successfully: %s\n", backupPath)

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	err = uploadToS3(ctx, s3Endpoint, s3AccessKey, s3SecretKey, s3BucketName, backupPath, fmt.Sprintf("%s-%s-backup.sql", dbName, dateNow))
	if err != nil {
		log.Fatalf("Failed to upload backup to MinIO: %v", err)
	}
	log.Println("Backup successfully loaded to MinIO")
}

func uploadToS3(ctx context.Context, endpoint, accessKey, secretKey, bucketName, filePath, objectKey string) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("Failure during AWS SDK configuration: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
		o.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	})

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failure during opening file: %w", err)
	}
	defer file.Close()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to load file to S3: %w", err)
	}

	return nil
}
