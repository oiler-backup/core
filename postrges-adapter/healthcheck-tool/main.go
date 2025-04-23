package main

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// Получение переменных окружения
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	// maxAttempts := os.Getenv("MAX_ATTEMPTS")
	// attemptInterval := os.Getenv("ATTEMPT_INTERVAL")

	// Преобразование переменных
	maxAttemptsInt := 3
	attemptIntervalDuration := 5 * time.Minute

	// Счетчик неудачных попыток
	failCount := 0

	for {
		// Выполнение pg_amcheck
		cmd := fmt.Sprintf("pg_amcheck -d %s -h %s -p %s -U %s", dbName, dbHost, dbPort, dbUser)
		err := runCommand(cmd)
		if err != nil {
			fmt.Println("Database check failed:", err)
			failCount++
		} else {
			fmt.Println("Database is healthy.")
			break
		}

		// Проверка количества неудачных попыток
		if failCount >= maxAttemptsInt {
			fmt.Println("Max attempts reached. Triggering restore...")
			triggerRestore(dbHost, dbPort, dbUser, dbPassword, dbName)
			break
		}

		// Ожидание перед следующей попыткой
		time.Sleep(attemptIntervalDuration)
	}
}

func runCommand(cmd string) error {
	// Здесь выполняется команда pg_amcheck
	// Пример: exec.Command("bash", "-c", cmd).Run()
	return nil // Заменить на реальную реализацию
}

func triggerRestore(dbHost, dbPort, dbUser, dbPassword, dbName string) {
	// Создание Kubernetes клиента
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("Error creating in-cluster config:", err)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating Kubernetes client:", err)
		return
	}

	// Создание объекта BackupRestore
	restore := &BackupRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("restore-%s", dbName),
			Namespace: "default", // Указать нужный namespace
		},
		Spec: BackupRestoreSpec{
			DatabaseURI:    dbHost,
			DatabasePort:   dbPort,
			DatabaseUser:   dbUser,
			DatabasePass:   dbPassword,
			DatabaseName:   dbName,
			DatabaseType:   "postgres",
			S3Endpoint:     "http://minio-service:9000",
			S3AccessKey:    "access-key",
			S3SecretKey:    "secret-key",
			S3BucketName:   "backups",
			BackupRevision: "latest",
		},
	}

	// Отправка объекта в Kubernetes
	_, err = clientset.RESTClient().Post().
		Resource("backuprestores").
		Namespace("default").
		Body(restore).
		Do(context.TODO()).Get()
	if err != nil {
		fmt.Println("Error creating BackupRestore object:", err)
		return
	}

	fmt.Println("BackupRestore object created successfully.")
}
