package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func main() {
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	s3BackupDir := os.Getenv("S3_BACKUP_DIR")
	maxBackupCountStr := os.Getenv("MAX_BACKUP_COUNT")

	if s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3BucketName == "" || s3BackupDir == "" || maxBackupCountStr == "" {
		log.Fatal("Envs S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET_NAME, S3_BACKUP_DIR, MAX_BACKUP_COUNT are required")
	}

	maxBackupCount, err := strconv.Atoi(maxBackupCountStr)
	if err != nil || maxBackupCount < 1 {
		log.Fatalf("Incorrect value MAX_BACKUP_COUNT: %s", maxBackupCountStr)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     s3AccessKey,
				SecretAccessKey: s3SecretKey,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("Failure during AWS SDK configuration: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(s3Endpoint)
		o.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	})

	listOutput, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(s3BucketName),
		Prefix: aws.String(s3BackupDir),
	})
	if err != nil {
		log.Fatalf("Ошибка при получении списка объектов: %v", err)
	}

	objects := listOutput.Contents
	log.Printf("Found %d objects with prefix %s\n", len(objects), s3BackupDir)

	if len(objects) > maxBackupCount {
		sort.Slice(objects, func(i, j int) bool {
			return objects[i].LastModified.Before(*objects[j].LastModified)
		})

		toDelete := objects[:len(objects)-maxBackupCount]
		log.Printf("Starting deletion of %d objects\n", len(toDelete))

		deleteObjects := []types.ObjectIdentifier{}
		for _, obj := range toDelete {
			log.Printf("Deleting: %s\n", *obj.Key)
			deleteObjects = append(deleteObjects, types.ObjectIdentifier{Key: obj.Key})
		}

		_, err = client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(s3BucketName),
			Delete: &types.Delete{
				Objects: deleteObjects,
			},
		})
		if err != nil {
			log.Fatalf("Failure during objects deletion: %v", err)
		}

		log.Println("Objects are deleted successfully")
	} else {
		log.Println("Limit is not reached, skipping...")
	}
}
