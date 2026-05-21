package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server       ServerConfig
	AuthService  AuthServiceConfig
	UsersService UsersServiceConfig
	CORS         CORSConfig
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

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
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

	corsConfig, err := loadCORSConfig()
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
		CORS: corsConfig,
	}, nil
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}
}

func (cfg CORSConfig) WithDefaults() CORSConfig {
	defaults := DefaultCORSConfig()

	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = defaults.AllowedOrigins
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = defaults.AllowedMethods
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = defaults.AllowedHeaders
	}

	return cfg
}

func (cfg CORSConfig) Validate() error {
	if err := validateStringList("CORS_ALLOWED_ORIGINS", cfg.AllowedOrigins); err != nil {
		return err
	}
	if err := validateStringList("CORS_ALLOWED_METHODS", cfg.AllowedMethods); err != nil {
		return err
	}
	if err := validateStringList("CORS_ALLOWED_HEADERS", cfg.AllowedHeaders); err != nil {
		return err
	}

	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("env var CORS_ALLOWED_ORIGINS cannot contain wildcard origin when credentials are allowed")
		}
	}

	return nil
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

func loadCORSConfig() (CORSConfig, error) {
	defaults := DefaultCORSConfig()

	allowedOrigins, err := getStringListEnv("CORS_ALLOWED_ORIGINS", defaults.AllowedOrigins)
	if err != nil {
		return CORSConfig{}, err
	}

	allowedMethods, err := getStringListEnv("CORS_ALLOWED_METHODS", defaults.AllowedMethods)
	if err != nil {
		return CORSConfig{}, err
	}

	allowedHeaders, err := getStringListEnv("CORS_ALLOWED_HEADERS", defaults.AllowedHeaders)
	if err != nil {
		return CORSConfig{}, err
	}

	cfg := CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: allowedMethods,
		AllowedHeaders: allowedHeaders,
	}
	if err := cfg.Validate(); err != nil {
		return CORSConfig{}, err
	}

	return cfg, nil
}

func getStringListEnv(key string, defaultValues []string) ([]string, error) {
	val := os.Getenv(key)
	if val == "" {
		return cloneStrings(defaultValues), nil
	}

	parts := strings.Split(val, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, fmt.Errorf("env var %s contains empty value", key)
		}
		values = append(values, item)
	}

	return values, nil
}

func validateStringList(key string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("env var %s must contain at least one value", key)
	}

	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("env var %s contains empty value", key)
		}
	}

	return nil
}

func cloneStrings(values []string) []string {
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}
