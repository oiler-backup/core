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
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	// Загрузка переменных окружения
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

	backupRevisionStr := os.Getenv("BACKUP_REVISION") // Параметр для выбора версии бэкапа
	backupRevision, err := strconv.Atoi(backupRevisionStr)
	if err != nil || backupRevision < 0 {
		log.Fatalf("Invalid BACKUP_REVISION value: %s. It must be a non-negative integer.", backupRevisionStr)
	}

	// 1. Получение списка бэкапов из S3
	client, err := createS3Client(s3Endpoint, s3AccessKey, s3SecretKey)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	backupKeys, err := listBackupFiles(client, s3BucketName)
	if err != nil {
		log.Fatalf("Failed to list backup files from S3: %v", err)
	}

	// Сортировка бэкапов по времени (по убыванию)
	sort.Sort(sort.Reverse(sort.StringSlice(backupKeys)))

	// Выбор бэкапа на основе backupRevision
	if backupRevision >= len(backupKeys) {
		log.Fatalf("BACKUP_REVISION (%d) is out of range. Available backups: %d", backupRevision, len(backupKeys))
	}
	selectedBackupKey := backupKeys[backupRevision]

	// 2. Скачивание выбранного бэкапа из S3
	err = downloadBackupFromS3(client, s3BucketName, selectedBackupKey, backupPath)
	if err != nil {
		log.Fatalf("Failed to download backup from S3: %v", err)
	}
	fmt.Println("Backup successfully downloaded from S3:", selectedBackupKey)

	// 3. Восстановление бэкапа в PostgreSQL
	err = restorePostgresBackup(dbHost, dbPort, dbUser, dbPassword, dbName, backupPath)
	if err != nil {
		log.Fatalf("Failed to restore PostgreSQL backup: %v", err)
	}
	fmt.Println("PostgreSQL backup successfully restored")
}

// createS3Client создает клиента S3
func createS3Client(endpoint, accessKey, secretKey string) (*s3.Client, error) {
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == s3.ServiceID && region == "us-east-1" { // Настройте регион по необходимости
			return aws.Endpoint{
				URL: endpoint,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"), // Убедитесь, что регион соответствует вашему S3
		config.WithEndpointResolver(customResolver),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	})

	return client, nil
}

// listBackupFiles получает список файлов бэкапов из S3
func listBackupFiles(client *s3.Client, bucketName string) ([]string, error) {
	var backupKeys []string

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3 bucket: %v", err)
		}

		for _, obj := range page.Contents {
			backupKeys = append(backupKeys, *obj.Key)
		}
	}

	return backupKeys, nil
}

// downloadBackupFromS3 скачивает выбранный бэкап из S3 в локальный файл
func downloadBackupFromS3(client *s3.Client, bucketName, objectKey, localFilePath string) error {
	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to get S3 object: %v", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer file.Close()

	_, err = file.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write S3 object to local file: %v", err)
	}

	return nil
}

// restorePostgresBackup выполняет восстановление бэкапа в PostgreSQL
func restorePostgresBackup(host, port, user, password, dbName, backupPath string) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connection to database successful")

	cmd := exec.Command("pg_restore",
		"-h", host,
		"-p", port,
		"-U", user,
		"-d", dbName,
		"--no-owner",
		"--clean",
		backupPath,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed executing pg_restore: %v\n%s", err, output)
		return err
	}
	log.Printf("Backup have been restored successfully: %s\n", backupPath)

	return nil
}
