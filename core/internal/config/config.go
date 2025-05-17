package config

import "github.com/caarlos0/env/v11"

type Config struct {
	OperatorNamespace string `env:"OPERATOR_NAMESPACE" envDefault:"oiler-backup-system"`
}

func GetConfig() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
