package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetConfig_Success(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("CORE_ADDR", "http://core:8080")
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_ACCESS_KEY", "access_key")
	t.Setenv("S3_SECRET_KEY", "secret_key")
	t.Setenv("S3_BUCKET_NAME", "backup-bucket")
	t.Setenv("MAX_BACKUP_COUNT", "5")
	t.Setenv("SECURE", "true")

	cfg, err := GetConfig()
	require.NoError(t, err)

	expected := Config{
		DbHost:         "localhost",
		DbPort:         "5432",
		DbUser:         "user",
		DbPassword:     "pass",
		DbName:         "mydb",
		CoreAddr:       "http://core:8080",
		S3Endpoint:     "s3.example.com",
		S3AccessKey:    "access_key",
		S3SecretKey:    "secret_key",
		S3BucketName:   "backup-bucket",
		MaxBackupCount: 5,
		Secure:         true,
	}

	assert.Equal(t, expected, cfg)
}

func Test_GetConfig_MissingRequiredField(t *testing.T) {
	os.Clearenv()
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("CORE_ADDR", "http://core:8080")
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_ACCESS_KEY", "access_key")
	t.Setenv("S3_SECRET_KEY", "secret_key")
	t.Setenv("S3_BUCKET_NAME", "backup-bucket")

	_, err := GetConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_HOST")
}

func Test_GetConfig_EmptyValue(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("CORE_ADDR", "http://core:8080")
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_ACCESS_KEY", "access_key")
	t.Setenv("S3_SECRET_KEY", "secret_key")
	t.Setenv("S3_BUCKET_NAME", "backup-bucket")

	_, err := GetConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_HOST")
}

func Test_GetConfig_DefaultValues(t *testing.T) {
	os.Clearenv()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("CORE_ADDR", "http://core:8080")
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_ACCESS_KEY", "access_key")
	t.Setenv("S3_SECRET_KEY", "secret_key")
	t.Setenv("S3_BUCKET_NAME", "backup-bucket")

	cfg, err := GetConfig()
	require.NoError(t, err)

	assert.Equal(t, 0, cfg.MaxBackupCount)
	assert.False(t, cfg.Secure)
}

func Test_String(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("CORE_ADDR", "http://core:8080")
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_ACCESS_KEY", "access_key")
	t.Setenv("S3_SECRET_KEY", "secret_key")
	t.Setenv("S3_BUCKET_NAME", "backup-bucket")
	t.Setenv("MAX_BACKUP_COUNT", "5")
	t.Setenv("SECURE", "true")

	cfg, err := GetConfig()
	require.NoError(t, err)

	expected := "{DbHost: localhost, DbPort: 5432, DbUser: user, DbPassword: <unset>, " +
		"DbName: mydb, CoreAddr: http://core:8080, S3Endpoint: s3.example.com, S3AccessKey: <unset>, " +
		"S3SecretKey: <unset>, S3BucketName: backup-bucket, MaxBackupCount: 5, Secure: true}"
	assert.Equal(t, expected, cfg.String())

}
