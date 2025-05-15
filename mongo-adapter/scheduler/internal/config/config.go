package config

import "github.com/caarlos0/env/v11"

type Config struct {
	SystemNamespace string `env:"SYSTEM_NAMESPACE,required"`
	BackuperVersion string `env:"BACKUPER_VERSION" envDefault:"ashadrinnn/mongobackuper:0.0.1-0"`
	RestorerVersion string `env:"RESTORER_VERSION" envDefault:"sveb00/mongorestorer:0.0.1-1"`
	Port            int64  `env:"PORT" envDefault:"50051"`
}

func GetConfig() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
