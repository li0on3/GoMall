package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// CreateDatabase 创建数据库
func CreateDatabase(config *Config) error {
	// 连接到MySQL服务器（不指定数据库）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.DBUser,
		config.DBPassword,
		config.DBHost,
		config.DBPort)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接MySQL服务器失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		return fmt.Errorf("无法ping通MySQL服务器: %v", err)
	}

	// 创建数据库
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", config.DBName)
	_, err = db.Exec(createDBSQL)
	if err != nil {
		return fmt.Errorf("创建数据库失败: %v", err)
	}

	log.Printf("数据库 %s 创建成功或已存在", config.DBName)
	return nil
}