package config

import (
	"os"

	"github.com/joho/godotenv"
)

type appConfig struct {
	Port string
}

type dbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type Config struct {
	App appConfig
	DB  dbConfig
}

func Init(path string) (*Config, error) {
	if err := godotenv.Load(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var dbName string

	switch getEnv("APP_ENV", "") {
	case "test":
		dbName = getEnv("DB_TEST_NAME", "dbname_test")
	default:
		dbName = getEnv("DB_NAME", "dbname")
	}

	return &Config{
		App: appConfig{
			Port: getEnv("APP_PORT", "5000"),
		},
		DB: dbConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "test"),
			Password: getEnv("DB_PASSWORD", "test"),
			Name:     dbName,
		},
	}, nil
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
