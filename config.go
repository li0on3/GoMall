package main

import (
	"os"
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
	RedisHost string
	RedisPort string

	// JWT配置
	JWTSecret string

	// 服务器配置
	ServerPort string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	config := &Config{
		// 数据库配置，提供默认值
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "gomall"),

		// Redis配置
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// JWT密钥
		JWTSecret: getEnv("JWT_SECRET", "gomall_jwt_secret_key_2024"),

		// 服务器端口
		ServerPort: getEnv("SERVER_PORT", "8080"),
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