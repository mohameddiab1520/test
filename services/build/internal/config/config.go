package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                int
	Environment         string
	LogLevel            string
	DBHost              string
	DBPort              int
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	RedisHost           string
	RedisPort           int
	RedisPassword       string
	RedisDB             int
	S3Endpoint          string
	S3AccessKey         string
	S3SecretKey         string
	S3Bucket            string
	S3Region            string
	S3UseSSL            bool
	UnityPath           string
	WorkspaceDir        string
	DefaultUnityVersion string
	BuildArtifactsPath  string
	AWSRegion           string
	MaxConcurrentBuilds int
	BuildTimeout        int
}

func Load() *Config {
	return &Config{
		Port:                getEnvAsInt("PORT", 8087),
		Environment:         getEnv("ENVIRONMENT", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnvAsInt("DB_PORT", 5432),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "unity_collab"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvAsInt("REDIS_DB", 0),
		S3Endpoint:          getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:         getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:         getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:            getEnv("S3_BUCKET", "unity-collab-builds"),
		S3Region:            getEnv("S3_REGION", "us-east-1"),
		S3UseSSL:            getEnvAsBool("S3_USE_SSL", false),
		UnityPath:           getEnv("UNITY_PATH", "/Applications/Unity/Hub/Editor"),
		WorkspaceDir:        getEnv("WORKSPACE_DIR", "./workspace"),
		DefaultUnityVersion: getEnv("DEFAULT_UNITY_VERSION", "2022.3.0f1"),
		BuildArtifactsPath:  getEnv("BUILD_ARTIFACTS_PATH", "./build-artifacts"),
		AWSRegion:           getEnv("AWS_REGION", "us-east-1"),
		MaxConcurrentBuilds: getEnvAsInt("MAX_CONCURRENT_BUILDS", 3),
		BuildTimeout:        getEnvAsInt("BUILD_TIMEOUT", 3600),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
