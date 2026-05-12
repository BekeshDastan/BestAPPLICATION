package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	GRPC     GRPCConfig
	SMTP     SMTPConfig
}

type PostgresConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
}

type NATSConfig struct {
	URL    string
	Stream string
}

type GRPCConfig struct {
	Port string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

func Load() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", "5432")
	viper.SetDefault("POSTGRES_SSLMODE", "disable")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("NATS_URL", "nats://localhost:4222")
	viper.SetDefault("NATS_STREAM", "SOCIAL")
	viper.SetDefault("NOTIFICATION_GRPC_PORT", "50055")
	viper.SetDefault("SMTP_HOST", "smtp.gmail.com")
	viper.SetDefault("SMTP_PORT", "587")

	cfg := &Config{}

	cfg.Postgres.DSN = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=notification_db sslmode=%s",
		viper.GetString("POSTGRES_HOST"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_SSLMODE"),
	)

	cfg.Redis.Addr = viper.GetString("REDIS_ADDR")
	cfg.Redis.Password = viper.GetString("REDIS_PASSWORD")

	cfg.NATS.URL = viper.GetString("NATS_URL")
	cfg.NATS.Stream = viper.GetString("NATS_STREAM")

	cfg.GRPC.Port = viper.GetString("NOTIFICATION_GRPC_PORT")

	cfg.SMTP.Host = viper.GetString("SMTP_HOST")
	cfg.SMTP.Port = viper.GetString("SMTP_PORT")
	cfg.SMTP.Username = viper.GetString("SMTP_USERNAME")
	cfg.SMTP.Password = viper.GetString("SMTP_PASSWORD")

	return cfg, nil
}
