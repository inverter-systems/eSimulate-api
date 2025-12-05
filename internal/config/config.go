package config

import (
	"esimulate-backend/internal/logger"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	AdminEmail  string
	AdminPassword string
	LogLevel    string
}

func LoadConfig() *Config {
	// Tenta carregar do .env, mas não falha se não existir (ambiente prod pode usar vars reais)
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://esimulate:DbaInv=2025@localhost:5432/esimulate_v1?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "change_this_secret_in_production_please"),
		AdminEmail:   getEnv("ADMIN_EMAIL", "admin@esimulate.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		LogLevel:     getEnv("LOG_LEVEL", "INFO"),
	}

	// Inicializar logger com o nível configurado
	logger.InitLogger(cfg.LogLevel)

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	// Apenas logar em DEBUG para não poluir logs em produção
	logger.Debug("Config: %s not found, using default.", key)
	return fallback
}
