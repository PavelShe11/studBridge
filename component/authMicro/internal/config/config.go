package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DB                     DBConfig
	JWT                    JWTConfig
	HttpServerAddr         string
	AccountServiceGrpcAddr string
}

type HTTPConfig struct {
	ServerAddr string
}

type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
	Secret                 string
}

func NewConfig() (*Config, []error) {
	errors := make([]error, 0)
	return &Config{
		DB: DBConfig{
			Host:     getEnvIsRequiredWithErrors("DBHost", &errors),
			Port:     getEnv("DBPort", "5432"),
			Username: getEnv("DBUsername", "db"),
			Password: getEnv("DBPassword", "db"),
			DBName:   getEnvIsRequiredWithErrors("DBName", &errors),
			SSLMode:  getEnv("DBSSLMode", "disable"),
		},
		JWT: JWTConfig{
			AccessTokenExpiration:  time.Duration(getEnvIsRequiredWithErrorsAsInt("JWTAccessTokenExpiration", &errors)) * time.Millisecond,
			RefreshTokenExpiration: time.Duration(getEnvIsRequiredWithErrorsAsInt("JWTRefreshTokenExpiration", &errors)) * time.Millisecond,
			Secret:                 getEnvIsRequiredWithErrors("JWTSecret", &errors),
		},
		HttpServerAddr:         getEnv("HttpServerAddr", "0.0.0.0:80"),
		AccountServiceGrpcAddr: getEnvIsRequiredWithErrors("AccountServiceGrpcAddr", &errors),
	}, errors
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvIsRequired(key string) (string, error) {
	if value, exists := os.LookupEnv(key); exists {
		return value, nil
	}
	return "", fmt.Errorf("missing required environment variable: %s", key)
}

func getEnvIsRequiredWithErrors(key string, errors *[]error) string {
	result, e := getEnvIsRequired(key)
	if e != nil {
		*errors = append(*errors, e)
		return ""
	}
	return result
}

func getEnvIsRequiredWithErrorsAsInt(key string, errors *[]error) int {
	resString, e := getEnvIsRequired(key)
	if e != nil {
		*errors = append(*errors, e)
		return 0
	}
	result, err := strconv.Atoi(resString)
	if err != nil {
		*errors = append(*errors, err)
		return 0
	}
	return result
}
