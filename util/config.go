package util

import "github.com/spf13/viper"

// Config holds all configuration of the application
// the values of Config are loaded from the environment variables or a config file
type Config struct {
	PORT           		string `mapstructure:"PORT"`
	DB_URL				string `mapstructure:"DB_URL"`
	TESTING_DB_URL		string `mapstructure:"TESTING_DB_URL"`
	SERVER_ADDRESS 		string `mapstructure:"SERVER_ADDRESS"`
}


func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}
	return
}
