package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	APIFootballKey    string
	APIFootballHost   string
	APIFootballBase   string
	APIFootballAuth   string // apisports (direct) or rapidapi
	SyncInterval      time.Duration
	LiveInterval      time.Duration
	BackfillStartYear int
	LiveScope         string // top5 or all
	LiveMaxFixtures   int
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	syncInterval := 6 * time.Hour
	if v := os.Getenv("COLLECTOR_SYNC_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			syncInterval = d
		}
	}

	liveInterval := 60 * time.Second
	if v := os.Getenv("COLLECTOR_LIVE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			liveInterval = d
		}
	}

	backfillStart := 2018
	if v := os.Getenv("COLLECTOR_BACKFILL_START_YEAR"); v != "" {
		if y := parseInt(v); y > 0 {
			backfillStart = y
		}
	}

	liveScope := getEnv("COLLECTOR_LIVE_SCOPE", "all")
	liveMax := 20
	if v := os.Getenv("COLLECTOR_LIVE_MAX"); v != "" {
		if n := parseInt(v); n > 0 {
			liveMax = n
		}
	}

	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://prono:prono_secret@localhost:5432/prono?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		APIFootballKey:    os.Getenv("API_FOOTBALL_KEY"),
		APIFootballHost:   getEnv("API_FOOTBALL_HOST", "v3.football.api-sports.io"),
		APIFootballBase:   getEnv("API_FOOTBALL_BASE_URL", "https://v3.football.api-sports.io"),
		APIFootballAuth:   getEnv("API_FOOTBALL_AUTH", "apisports"),
		SyncInterval:      syncInterval,
		LiveInterval:      liveInterval,
		BackfillStartYear: backfillStart,
		LiveScope:         liveScope,
		LiveMaxFixtures:   liveMax,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// Top5LeagueIDs API-Football external IDs for Top 5 European leagues
var Top5LeagueIDs = []int{39, 140, 135, 78, 61}
