package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type CodeGenConfig struct {
	CodeTTL       time.Duration
	CodePattern   string
	CodeMaxLength int
}

type Config struct {
	DB                     DBConfig
	JWT                    JWTConfig
	HttpServerAddr         string
	AccountServiceGrpcAddr string
	CodeGenConfig          CodeGenConfig
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
			Username: getEnv("DBUsername", "postgres"),
			Password: getEnv("DBPassword", "postgres"),
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
		CodeGenConfig: CodeGenConfig{
			CodeTTL:       time.Duration(getEnvAsInt("CodeTTL", 15)) * time.Minute,
			CodePattern:   getEnv("CodePattern", "\\d{6}"),
			CodeMaxLength: getEnvAsInt("CodeMaxLength", 6),
		},
	}, errors
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt читает переменную окружения как int с дефолтом при пустом значении или ошибке парсинга.
func getEnvAsInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	res, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return res
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
