package configs

import (
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type AppConfig struct {
	Database  DatabaseConfig
	Migration MigrationConfig
	Redis     RedisConfig
	GRPC      GrpcConfig
	Server    ServerConfig
	RabbitMQ  struct {
		URL string `env:"RABBITMQ_URL,required"`
	}
}

func LoadConfig(log *logrus.Logger) (*AppConfig, error) {
	if os.Getenv("ENV") != "production" {
		log.Info("ENV not production")
		if err := godotenv.Load(); err != nil {
			log.Fatalf("FATAL: Gagal memuat file .env. Pastikan file ada. Error: %v", err)
		}
		log.Info("Berhasil memuat konfigurasi dari file .env (Mode Development)")
	}
	cfg := &AppConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	log.Info("Konfigurasi terstruktur berhasil dimuat.")
	return cfg, nil
}
