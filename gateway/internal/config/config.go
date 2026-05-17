package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	HTTP        HTTPConfig
	GRPC        GRPCTargets
	MinIO       MinIOConfig
	AdminEmails []string
}

type HTTPConfig struct {
	Port          string
	AllowedOrigin string // e.g. "http://localhost:3000". "*" disables credentials.
}

type GRPCTargets struct {
	User         string
	Post         string
	Chat         string
	Story        string
	Notification string
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	PublicHost string
	UseSSL     bool
}

func Load() *Config {
	viper.AutomaticEnv()
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("USER_GRPC_ADDR", "localhost:50051")
	viper.SetDefault("POST_GRPC_ADDR", "localhost:50052")
	viper.SetDefault("CHAT_GRPC_ADDR", "localhost:50053")
	viper.SetDefault("STORY_GRPC_ADDR", "localhost:50054")
	viper.SetDefault("NOTIFICATION_GRPC_ADDR", "localhost:50055")
	viper.SetDefault("MINIO_ENDPOINT", "minio:9000")
	viper.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO_SECRET_KEY", "minioadmin")
	viper.SetDefault("MINIO_BUCKET", "social")
	viper.SetDefault("MINIO_PUBLIC_HOST", "localhost:9000")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("ADMIN_EMAILS", "")
	viper.SetDefault("ALLOWED_ORIGIN", "http://localhost:3000")

	rawEmails := viper.GetString("ADMIN_EMAILS")
	emails := make([]string, 0)
	for _, e := range strings.Split(rawEmails, ",") {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" {
			emails = append(emails, e)
		}
	}

	return &Config{
		HTTP: HTTPConfig{
			Port:          viper.GetString("HTTP_PORT"),
			AllowedOrigin: viper.GetString("ALLOWED_ORIGIN"),
		},
		GRPC: GRPCTargets{
			User:         viper.GetString("USER_GRPC_ADDR"),
			Post:         viper.GetString("POST_GRPC_ADDR"),
			Chat:         viper.GetString("CHAT_GRPC_ADDR"),
			Story:        viper.GetString("STORY_GRPC_ADDR"),
			Notification: viper.GetString("NOTIFICATION_GRPC_ADDR"),
		},
		MinIO: MinIOConfig{
			Endpoint:   viper.GetString("MINIO_ENDPOINT"),
			AccessKey:  viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey:  viper.GetString("MINIO_SECRET_KEY"),
			Bucket:     viper.GetString("MINIO_BUCKET"),
			PublicHost: viper.GetString("MINIO_PUBLIC_HOST"),
			UseSSL:     viper.GetBool("MINIO_USE_SSL"),
		},
		AdminEmails: emails,
	}
}
