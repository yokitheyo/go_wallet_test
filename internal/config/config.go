package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DBHost   string
	DBPort   int
	DBUser   string
	DBPass   string
	DBName   string
	HTTPPort string
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("warning: cannot read config.yaml: %v\n", err)
	}

	v.BindEnv("db.host", "DB_eHOST")
	v.BindEnv("db.port", "DB_PORT")
	v.BindEnv("db.user", "DB_USER")
	v.BindEnv("db.pass", "DB_PASS")
	v.BindEnv("db.name", "DB_NAME")
	v.BindEnv("http.port", "HTTP_PORT")

	return &Config{
		DBHost:   v.GetString("db.host"),
		DBPort:   v.GetInt("db.port"),
		DBUser:   v.GetString("db.user"),
		DBPass:   v.GetString("db.pass"),
		DBName:   v.GetString("db.name"),
		HTTPPort: v.GetString("http.port"),
	}, nil
}
