// Package config stores configuration for backuper.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// A Config stores configuraton.
type Config struct {
	DbHost       string `env:"DB_HOST,required,notEmpty"`
	DbPort       string `env:"DB_PORT,required,notEmpty"`
	DbUser       string `env:"DB_USER,required,notEmpty"`
	DbPassword   string `env:"DB_PASSWORD,required,notEmpty,unset"`
	DbName       string `env:"DB_NAME,required,notEmpty"`
	CoreAddr     string `env:"CORE_ADDR,required,notEmpty"` // Uri of an Kubernetes Operator core
	S3Endpoint   string `env:"S3_ENDPOINT,required,notEmpty"`
	S3AccessKey  string `env:"S3_ACCESS_KEY,required,notEmpty,unset"`
	S3SecretKey  string `env:"S3_SECRET_KEY,required,notEmpty,unset"`
	S3BucketName string `env:"S3_BUCKET_NAME,required,notEmpty"`

	MaxBackupCount int  `env:"MAX_BACKUP_COUNT"`
	Secure         bool `env:"SECURE" envDefault:"false"` // TLS/SSL Encryption
}

// GetConfig reads environment variables, validates them and return Config object or
// error if occured.
func GetConfig() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) String() string {
	return fmt.Sprintf("{DbHost: %s, DbPort: %s, DbUser: %s, DbPassword: <unset>, DbName: %s, "+
		"CoreAddr: %s, S3Endpoint: %s, S3AccessKey: <unset>, S3SecretKey: <unset>, S3BucketName: %s, "+
		"MaxBackupCount: %d, Secure: %t}",
		c.DbHost, c.DbPort, c.DbUser, c.DbName,
		c.CoreAddr, c.S3Endpoint, c.S3BucketName,
		c.MaxBackupCount, c.Secure)
}
