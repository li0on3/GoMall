package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置结构体
type Config struct {
	// 数据库配置
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis配置
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// JWT配置
	JWTSecret string

	// 服务器配置
	ServerPort string
	GinMode    string

	// 文件上传配置
	UploadPath       string
	MaxFileSize      int64
	AllowedFileTypes string

	// 缓存配置
	CacheDefaultExpiration int
	CacheCleanupInterval   int
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Printf("警告：无法加载.env文件: %v，将使用环境变量或默认值", err)
	}

	config := &Config{
		// 数据库配置
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "test"),
		DBPassword: getEnv("DB_PASSWORD", "!QAZzse4"),
		DBName:     getEnv("DB_NAME", "gomall"),

		// Redis配置
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// JWT密钥
		JWTSecret: getEnv("JWT_SECRET", "gomall_jwt_secret_key_2024_very_secure"),

		// 服务器配置
		ServerPort: getEnv("SERVER_PORT", "8080"),
		GinMode:    getEnv("GIN_MODE", "debug"),

		// 文件上传配置
		UploadPath:       getEnv("UPLOAD_PATH", "./upload"),
		MaxFileSize:      getEnvAsInt64("MAX_FILE_SIZE", 10485760), // 10MB
		AllowedFileTypes: getEnv("ALLOWED_FILE_TYPES", "jpg,jpeg,png,gif,txt,md,pdf,doc,docx"),

		// 缓存配置
		CacheDefaultExpiration: getEnvAsInt("CACHE_DEFAULT_EXPIRATION", 3600),   // 1小时
		CacheCleanupInterval:   getEnvAsInt("CACHE_CLEANUP_INTERVAL", 600),     // 10分钟
	}

	return config
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为int，如果不存在或转换失败则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsInt64 获取环境变量并转换为int64，如果不存在或转换失败则返回默认值
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}