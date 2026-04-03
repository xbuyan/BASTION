package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Env    string
	Port   string
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
	RedisURL    string
	JWTSecret   string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
	FrontendURL string
	DockerEnabled bool
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("ENV", "development")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "prds")
	viper.SetDefault("DB_NAME", "prds")
	viper.SetDefault("REDIS_URL", "localhost:6379")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("DOCKER_ENABLED", "true")

	_ = viper.ReadInConfig()

	cfg := &Config{
		Env:                viper.GetString("ENV"),
		Port:               viper.GetString("PORT"),
		DBHost:             viper.GetString("DB_HOST"),
		DBPort:             viper.GetString("DB_PORT"),
		DBUser:             viper.GetString("DB_USER"),
		DBPass:             viper.GetString("DB_PASSWORD"),
		DBName:             viper.GetString("DB_NAME"),
		RedisURL:           viper.GetString("REDIS_URL"),
		JWTSecret:          viper.GetString("JWT_SECRET"),
		GitHubClientID:     viper.GetString("GITHUB_CLIENT_ID"),
		GitHubClientSecret: viper.GetString("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  viper.GetString("GITHUB_CALLBACK_URL"),
		FrontendURL:        viper.GetString("FRONTEND_URL"),
		DockerEnabled:      viper.GetBool("DOCKER_ENABLED"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// DatabaseURL returns the full PostgreSQL connection string.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName,
	)
}

// IsDev returns true for non-production environments.
func (c *Config) IsDev() bool {
	return c.Env != "production"
}

// GetEnv is a helper for reading env vars with fallback.
func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
