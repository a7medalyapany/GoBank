package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration of the application
// the values of Config are loaded from the environment variables or a config file
type Config struct {
	PORT           		string 			`mapstructure:"PORT"`
	DB_URL				string 			`mapstructure:"DB_URL"`
	TESTING_DB_URL		string 			`mapstructure:"TESTING_DB_URL"`
	SERVER_ADDRESS 		string 			`mapstructure:"SERVER_ADDRESS"`
	TOKEN_SYMMETRIC_KEY	string 			`mapstructure:"TOKEN_SYMMETRIC_KEY"`
	ACCESS_TOKEN_DURATION time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
}


// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
    viper.AddConfigPath(path)
    viper.SetConfigName("app")
    viper.SetConfigType("env")
    viper.AutomaticEnv()

    // Only read file if it exists — in production, env vars are enough
    if err = viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return // real error, not just missing file
        }
        err = nil // file not found is fine in production
    }

    err = viper.Unmarshal(&config)
    return
}