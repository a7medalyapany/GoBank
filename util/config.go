package util

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration of the application
// the values of Config are loaded from the environment variables or a config file
type Config struct {
	PORT                   string        `mapstructure:"PORT"`
	DB_URL                 string        `mapstructure:"DB_URL"`
	TESTING_DB_URL         string        `mapstructure:"TESTING_DB_URL"`
	SERVER_ADDRESS         string        `mapstructure:"SERVER_ADDRESS"`
	TOKEN_SYMMETRIC_KEY    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	ACCESS_TOKEN_DURATION  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	REFRESH_TOKEN_DURATION time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	GRPC_SERVER_PORT       string        `mapstructure:"GRPC_SERVER_PORT"`
	ENVIRONMENT            string        `mapstructure:"ENVIRONMENT"`
    REDIS_ADDRESS          string        `mapstructure:"REDIS_ADDRESS"`
    REDIS_PASSWORD         string        `mapstructure:"REDIS_PASSWORD"`
    REDIS_TLS              bool          `mapstructure:"REDIS_TLS"`
    BASE_URL               string        `mapstructure:"BASE_URL"`
    EMAIL_SENDER_NAME      string        `mapstructure:"EMAIL_SENDER_NAME"`
    EMAIL_SENDER_ADDRESS   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
    EMAIL_SENDER_PASSWORD  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
}


// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (config Config, err error) {
    v := viper.New()
    
    // Try file first (local dev only)
    v.SetConfigFile("app.env")
    v.SetConfigType("env")
    _ = v.ReadInConfig()

    // This is what makes Unmarshal respect env vars
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Explicitly bind each key to its env var name
    keys := []string{
        "PORT", "DB_URL", "TESTING_DB_URL", "SERVER_ADDRESS",
        "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION",
        "GRPC_SERVER_PORT", "ENVIRONMENT", "REDIS_ADDRESS", "REDIS_PASSWORD",
        "REDIS_TLS", "BASE_URL", "EMAIL_SENDER_NAME",
        "EMAIL_SENDER_ADDRESS", "EMAIL_SENDER_PASSWORD",
    }
    for _, key := range keys {
        v.BindEnv(key)
    }

    err = v.Unmarshal(&config)
    return
}