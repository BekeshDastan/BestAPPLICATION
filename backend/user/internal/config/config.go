package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	GRPC     GRPCConfig
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

type JWTConfig struct {
	PrivateKey  *ecdsa.PrivateKey
	PublicKey   *ecdsa.PublicKey
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
}

type GRPCConfig struct {
	Port string
}

func Load() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", "5432")
	viper.SetDefault("POSTGRES_SSLMODE", "disable")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("NATS_URL", "nats://localhost:4222")
	viper.SetDefault("NATS_STREAM", "SOCIAL")
	viper.SetDefault("USER_GRPC_PORT", "50051")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")

	cfg := &Config{}

	cfg.Postgres.DSN = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=user_db sslmode=%s",
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

	cfg.GRPC.Port = viper.GetString("USER_GRPC_PORT")

	privKey, pubKey, err := loadECDSAKeys(
		viper.GetString("JWT_PRIVATE_KEY_PATH"),
		viper.GetString("JWT_PUBLIC_KEY_PATH"),
	)
	if err != nil {
		return nil, fmt.Errorf("load jwt keys: %w", err)
	}
	cfg.JWT.PrivateKey = privKey
	cfg.JWT.PublicKey = pubKey

	if d, err := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL")); err == nil {
		cfg.JWT.AccessTTL = d
	} else {
		cfg.JWT.AccessTTL = 15 * time.Minute
	}
	if d, err := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL")); err == nil {
		cfg.JWT.RefreshTTL = d
	} else {
		cfg.JWT.RefreshTTL = 7 * 24 * time.Hour
	}

	return cfg, nil
}

func loadECDSAKeys(privPath, pubPath string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key file %q: %w", privPath, err)
	}
	block, _ := pem.Decode(privBytes)
	if block == nil {
		return nil, nil, fmt.Errorf("decode private key PEM: invalid format")
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse EC private key: %w", err)
	}

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read public key file %q: %w", pubPath, err)
	}
	block, _ = pem.Decode(pubBytes)
	if block == nil {
		return nil, nil, fmt.Errorf("decode public key PEM: invalid format")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse public key: %w", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("public key is not ECDSA")
	}
	return privKey, ecPub, nil
}
