package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	AppConfig = LoadConfig()
	
	// 创建数据库
	if err := CreateDatabase(AppConfig); err != nil {
		log.Fatalf("数据库创建失败: %v", err)
	}
	
	// 初始化数据库连接
	if err := InitDatabase(AppConfig); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	
	// 初始化Redis连接
	if err := InitRedis(AppConfig); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}
	
	// 确保程序退出时关闭数据库连接
	defer CloseDatabase()
	
	// 设置Gin模式
	gin.SetMode(gin.DebugMode)
	
	// 创建Gin引擎
	r := gin.Default()
	
	// 加载HTML模板
	r.LoadHTMLGlob("templates/*")
	
	// 设置静态文件路由
	r.Static("/public", "./public")
	r.Static("/upload", "./upload")
	
	// 基础路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "GoMall - Go语言电商平台",
		})
	})
	
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"message": "GoMall服务运行正常",
		})
	})
	
	// API路由组
	api := r.Group("/api")
	{
		// 用户相关API
		users := api.Group("/users")
		{
			users.POST("/register", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "用户注册功能待实现"})
			})
			users.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "用户登录功能待实现"})
			})
		}
		
		// 商品相关API
		products := api.Group("/products")
		{
			products.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "商品列表功能待实现"})
			})
			products.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "商品详情功能待实现"})
			})
		}
		
		// 订单相关API
		orders := api.Group("/orders")
		{
			orders.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "订单列表功能待实现"})
			})
			orders.POST("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "创建订单功能待实现"})
			})
		}
	}
	
	// 监听程序中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Println("正在关闭服务器...")
		CloseDatabase()
		os.Exit(0)
	}()
	
	// 启动服务器
	serverAddr := ":" + AppConfig.ServerPort
	fmt.Printf("GoMall服务器启动成功，访问地址: http://localhost%s\n", serverAddr)
	log.Printf("服务器监听端口: %s", AppConfig.ServerPort)
	
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}