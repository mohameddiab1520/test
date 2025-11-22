package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServiceName string
	ServicePort int
	Environment string

	// PostgreSQL
	PostgresHost     string
	PostgresPort     int
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresPoolMax  int
	PostgresPoolMin  int

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Presence settings
	PresenceTTL           int
	ActivityRetentionDays int
	StatusUpdateInterval  int
}

func Load() *Config {
	// Load .env file if exists
	godotenv.Load()

	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "presence-service"),
		ServicePort: getEnvInt("SERVICE_PORT", 8085),
		Environment: getEnv("ENVIRONMENT", "development"),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnvInt("POSTGRES_PORT", 5432),
		PostgresDB:       getEnv("POSTGRES_DB", "collab_dev"),
		PostgresUser:     getEnv("POSTGRES_USER", "dev"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "devpass"),
		PostgresPoolMax:  getEnvInt("POSTGRES_POOL_MAX_SIZE", 20),
		PostgresPoolMin:  getEnvInt("POSTGRES_POOL_MIN_SIZE", 10),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		PresenceTTL:           getEnvInt("PRESENCE_TTL", 300),
		ActivityRetentionDays: getEnvInt("ACTIVITY_RETENTION_DAYS", 7),
		StatusUpdateInterval:  getEnvInt("STATUS_UPDATE_INTERVAL", 30),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
