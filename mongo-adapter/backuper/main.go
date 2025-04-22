package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	backupPath := "/tmp/backup"

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3BucketName := os.Getenv("S3_BUCKET_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" ||
		s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3BucketName == "" {
		log.Fatal("Envs DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET_NAME are required")
	}

	dumpCmd := exec.CommandContext(ctx, "mongodump",
		"--host", dbHost,
		"--port", dbPort,
		"--username", dbUser,
		"--password", dbPassword,
		"--db", dbName,
		"--out", backupPath,
		"--authenticationDatabase", "admin",
	)

	output, err := dumpCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed executing mongodump: %v\n%s", err, string(output))
	}
	log.Printf("Backup created successfully: %s\n", backupPath)

	dateNow := time.Now().Format("2006-01-02-15-04-05")
	err = uploadToS3(ctx, s3Endpoint, s3AccessKey, s3SecretKey, s3BucketName, backupPath, fmt.Sprintf("%s-%s-backup", dbName, dateNow))
	if err != nil {
		log.Fatalf("Failed to upload backup to MinIO: %v", err)
	}
	log.Println("Backup successfully loaded to MinIO")
}

func uploadToS3(ctx context.Context, endpoint, accessKey, secretKey, bucketName, dirPath, objectKey string) error {
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

	err = filepath.Walk(dirPath, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %w", filePath, err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %w", filePath, err)
		}
		defer file.Close()

		relativePath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path for %q: %w", filePath, err)
		}

		s3Key := filepath.Join(objectKey, relativePath)

		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(s3Key),
			Body:   file,
		})
		if err != nil {
			return fmt.Errorf("failed to upload file %q to S3 with key %q: %w", filePath, s3Key, err)
		}

		fmt.Printf("Uploaded %q to S3 with key %q\n", filePath, s3Key)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error during directory walk: %w", err)
	}

	return nil
}
