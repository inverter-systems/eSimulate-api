package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	AdminEmail  string
	AdminPassword string
}

func LoadConfig() *Config {
	// Tenta carregar do .env, mas não falha se não existir (ambiente prod pode usar vars reais)
	_ = godotenv.Load()

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://esimulate:DbaInv=2025@localhost:5432/esimulate_v1?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "change_this_secret_in_production_please"),
		AdminEmail:   getEnv("ADMIN_EMAIL", "admin@esimulate.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Config: %s not found, using default.", key)
	return fallback
}
