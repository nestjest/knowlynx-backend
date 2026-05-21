package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server       ServerConfig
	AuthService  AuthServiceConfig
	UsersService UsersServiceConfig
}

type ServerConfig struct {
	HttpAddr string
	RHT      time.Duration
}

type AuthServiceConfig struct {
	Addr string
}

type UsersServiceConfig struct {
	Addr string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not foun, using system env")
	}

	rht, err := parseDuration("RHT")
	if err != nil {
		return nil, err
	}

	authServiceAddr, err := requiredEnv("AUTH_SERVICE_ADDR")
	if err != nil {
		return nil, err
	}

	usersServiceAddr, err := requiredEnv("USERS_SERVICE_ADDR")
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			HttpAddr: getEnv("SERVER_ADDR", ":8080"),
			RHT:      rht,
		},
		AuthService: AuthServiceConfig{
			Addr: authServiceAddr,
		},
		UsersService: UsersServiceConfig{
			Addr: usersServiceAddr,
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func requiredEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("env var %s is required but not set", key)
	}
	return val, nil
}

func parseDuration(key string) (time.Duration, error) {
	val, err := requiredEnv(key)
	if err != nil {
		return 0, err
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}

	return d, nil
}
