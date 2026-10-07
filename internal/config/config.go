package config

import "os"

const DefaultJWTSecret = "dev-secret-change-me"

type Config struct {
	Port        string
	DatabaseURL string // empty means in-memory storage
	JWTSecret   string
}

func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   getenv("JWT_SECRET", DefaultJWTSecret),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
