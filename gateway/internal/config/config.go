package config

import (
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

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not foun, using system env")
	}

	return &Config{
		Server: ServerConfig{
			HttpAddr: getEnv("SERVER_ADDR", ":8080"),
			RHT:      mustParseDuration("RHT"),
		},
		AuthService: AuthServiceConfig{
			Addr: mustGetEnv("AUTH_SERVICE_ADDR"),
		},
		UsersService: UsersServiceConfig{
			Addr: mustGetEnv("USERS_SERVICE_ADDR"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("env var %s is required but not set", key)
	}
	return val
}

func mustParseDuration(key string) time.Duration {
	val := mustGetEnv(key)
	d, err := time.ParseDuration(val)

	if err != nil {
		log.Fatalf("invalid duration for %s: %v", key, err)
	}

	return d
}
