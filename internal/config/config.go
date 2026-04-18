package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret           string
	JWTAccessExpMinutes int
	JWTRefreshExpDays   int

	// Google OAuth
	GoogleClientID string
}

var App *Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	accessExp, _ := strconv.Atoi(getEnv("JWT_ACCESS_EXP_MINUTES", "15"))
	refreshExp, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXP_DAYS", "7"))

	App = &Config{
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "myuser"),
		DBPassword:          getEnv("DB_PASSWORD", "mypassword"),
		DBName:              getEnv("DB_NAME", "new_pos_db"),
		DBSSLMode:           getEnv("DB_SSL_MODE", "disable"),
		JWTSecret:           getEnv("JWT_SECRET", "JWTS3CRETK3Y15UN1QU3"),
		JWTAccessExpMinutes: accessExp,
		JWTRefreshExpDays:   refreshExp,
		GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
