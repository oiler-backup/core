package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
	s3ObjectKey := os.Getenv("S3_OBJECT_KEY") // Путь к файлу бэкапа в S3

	// 1. Скачивание бэкапа из S3
	err := downloadBackupFromS3(s3Endpoint, s3AccessKey, s3SecretKey, s3BucketName, s3ObjectKey, backupPath)
	if err != nil {
		log.Fatalf("Failed to download backup from S3: %v", err)
	}
	fmt.Println("Backup successfully downloaded from S3")

	// 2. Восстановление бэкапа в PostgreSQL
	err = restorePostgresBackup(dbHost, dbPort, dbUser, dbPassword, dbName, backupPath)
	if err != nil {
		log.Fatalf("Failed to restore PostgreSQL backup: %v", err)
	}
	fmt.Println("PostgreSQL backup successfully restored")
}

// downloadBackupFromS3 скачивает бэкап из S3 в локальный файл
func downloadBackupFromS3(endpoint, accessKey, secretKey, bucketName, objectKey, localFilePath string) error {
	// Создаем настраиваемый resolver для S3
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == s3.ServiceID && region == "us-east-1" { // Настройте регион по необходимости
			return aws.Endpoint{
				URL: endpoint,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Настройка клиента S3 с использованием нового API
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"), // Убедитесь, что регион соответствует вашему S3
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolver(customResolver),
	)
	if err != nil {
		return fmt.Errorf("failed to load S3 config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	// Скачивание объекта
	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to get S3 object: %v", err)
	}
	defer resp.Body.Close()

	// Запись в локальный файл
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
	// Формирование команды pg_restore
	cmd := exec.Command("pg_restore",
		"-h", host,
		"-p", port,
		"-U", user,
		"-d", dbName,
		"--no-owner",
		"--clean",
		backupPath,
	)

	// Настройка переменной окружения PGPASSWORD для аутентификации
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	// Перехват вывода команды
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	// Выполнение команды
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %s, stderr: %s", err, errb.String())
	}

	fmt.Println("pg_restore output:", outb.String())
	return nil
}
