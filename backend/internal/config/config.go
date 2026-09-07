package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	Port          string
	MLServiceURL  string
	AIAgentURL    string
	CORSOrigins   []string
	JWTExpiration time.Duration
	AppEnv        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	origins := strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"), ",")
	appEnv := getEnv("APP_ENV", "development")
	jwtSecret := getEnv("JWT_SECRET", "change_me_in_production")

	if appEnv == "production" && (jwtSecret == "" || jwtSecret == "change_me_in_production") {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://prono:prono_secret@localhost:5432/prono?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:     jwtSecret,
		Port:          getEnv("BACKEND_PORT", "8080"),
		MLServiceURL:  getEnv("ML_SERVICE_URL", "http://localhost:5000"),
		AIAgentURL:    getEnv("AI_AGENT_URL", "http://localhost:5001"),
		CORSOrigins:   origins,
		JWTExpiration: 24 * time.Hour,
		AppEnv:        appEnv,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
