package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB    *gorm.DB
	RDB   *redis.Client
	CTX   = context.Background()
	AppConfig *Config
)

// User 用户模型
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Phone        string    `json:"phone"`
	RealName     string    `json:"real_name"`
	Avatar       string    `json:"avatar"`
	Status       int       `json:"status" gorm:"default:1"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Category 商品分类模型
type Category struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	ParentID    uint      `json:"parent_id" gorm:"default:0"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	Status      int       `json:"status" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at"`
}

// Product 商品模型
type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Price       float64   `json:"price" gorm:"not null"`
	Stock       int       `json:"stock" gorm:"default:0"`
	CategoryID  uint      `json:"category_id"`
	Category    Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Images      string    `json:"images" gorm:"type:json"`
	Status      int       `json:"status" gorm:"default:1"`
	SalesCount  int       `json:"sales_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CartItem 购物车项目模型
type CartItem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	ProductID uint      `json:"product_id" gorm:"not null"`
	Product   Product   `json:"product" gorm:"foreignKey:ProductID"`
	Quantity  int       `json:"quantity" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order 订单模型
type Order struct {
	ID              uint        `json:"id" gorm:"primaryKey"`
	UserID          uint        `json:"user_id" gorm:"not null"`
	User            User        `json:"user" gorm:"foreignKey:UserID"`
	OrderNo         string      `json:"order_no" gorm:"uniqueIndex;not null"`
	TotalAmount     float64     `json:"total_amount" gorm:"not null"`
	Status          string      `json:"status" gorm:"default:pending"`
	ShippingAddress string      `json:"shipping_address"`
	OrderItems      []OrderItem `json:"order_items" gorm:"foreignKey:OrderID"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// OrderItem 订单商品模型
type OrderItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	OrderID   uint    `json:"order_id" gorm:"not null"`
	ProductID uint    `json:"product_id" gorm:"not null"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
	Quantity  int     `json:"quantity" gorm:"not null"`
	Price     float64 `json:"price" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// InitDatabase 初始化数据库连接
func InitDatabase(config *Config) error {
	var err error

	// MySQL连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.DBUser,
		config.DBPassword,
		config.DBHost,
		config.DBPort,
		config.DBName)

	// 连接MySQL数据库
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("连接MySQL数据库失败: %v", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表结构
	err = AutoMigrate()
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	log.Println("MySQL数据库连接成功")
	return nil
}

// InitRedis 初始化Redis连接
func InitRedis(config *Config) error {
	addr := fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort)
	
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // 没有密码
		DB:       0,  // 使用默认数据库
	})

	// 测试连接
	_, err := RDB.Ping(CTX).Result()
	if err != nil {
		return fmt.Errorf("连接Redis失败: %v", err)
	}

	log.Println("Redis连接成功")
	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&Category{},
		&Product{},
		&CartItem{},
		&Order{},
		&OrderItem{},
	)
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() {
	if sqlDB, err := DB.DB(); err == nil {
		sqlDB.Close()
	}
	if RDB != nil {
		RDB.Close()
	}
	log.Println("数据库连接已关闭")
}