package config

import "github.com/spf13/viper"

type Config struct {
	HTTP HTTPConfig
	GRPC GRPCTargets
}

type HTTPConfig struct {
	Port string
}

type GRPCTargets struct {
	User  string
	Post  string
	Chat  string
	Story string
}

func Load() *Config {
	viper.AutomaticEnv()
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("USER_GRPC_ADDR", "localhost:50051")
	viper.SetDefault("POST_GRPC_ADDR", "localhost:50052")
	viper.SetDefault("CHAT_GRPC_ADDR", "localhost:50053")
	viper.SetDefault("STORY_GRPC_ADDR", "localhost:50054")

	return &Config{
		HTTP: HTTPConfig{Port: viper.GetString("HTTP_PORT")},
		GRPC: GRPCTargets{
			User:  viper.GetString("USER_GRPC_ADDR"),
			Post:  viper.GetString("POST_GRPC_ADDR"),
			Chat:  viper.GetString("CHAT_GRPC_ADDR"),
			Story: viper.GetString("STORY_GRPC_ADDR"),
		},
	}
}
