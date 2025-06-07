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
	
	// 初始化订单服务
	InitOrderService()
	
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
			users.POST("/register", UserRegister)                           // 用户注册
			users.POST("/login", UserLogin)                                 // 用户登录
			users.POST("/logout", RequireUser(), UserLogout)                // 用户登出
			users.GET("/profile", RequireUser(), GetUserProfile)            // 获取用户信息
			users.PUT("/profile", RequireUser(), UpdateUserProfile)         // 更新用户信息
			users.PUT("/password", RequireUser(), ChangePassword)           // 修改密码
		}
		
		// 商品相关API
		products := api.Group("/products")
		{
			products.GET("", GetProducts)                                     // 获取商品列表
			products.GET("/hot", GetHotProducts)                             // 获取热门商品
			products.GET("/search", SearchProducts)                          // 搜索商品
			products.GET("/:id", GetProduct)                                 // 获取商品详情
			products.POST("", RequireUser(), CreateProduct)                  // 创建商品
			products.PUT("/:id", RequireUser(), UpdateProduct)               // 更新商品
			products.DELETE("/:id", RequireUser(), DeleteProduct)            // 删除商品
		}

		// 商品分类API
		categories := api.Group("/categories")
		{
			categories.GET("", GetCategories)                                // 获取分类列表
			categories.GET("/:id", GetCategory)                              // 获取分类详情
			categories.POST("", RequireUser(), CreateCategory)               // 创建分类
			categories.PUT("/:id", RequireUser(), UpdateCategory)            // 更新分类
			categories.DELETE("/:id", RequireUser(), DeleteCategory)         // 删除分类
		}

		// 文件上传API
		upload := api.Group("/upload")
		{
			upload.POST("/images", RequireUser(), UploadProductImages)       // 上传商品图片
		}
		
		// 购物车相关API
		cart := api.Group("/cart")
		{
			cart.GET("", RequireUser(), GetCart)                               // 获取购物车
			cart.POST("/add", RequireUser(), AddToCart)                        // 添加到购物车
			cart.PUT("/:id", RequireUser(), UpdateCartItem)                    // 更新购物车项
			cart.DELETE("/:id", RequireUser(), DeleteCartItem)                 // 删除购物车项
			cart.DELETE("/clear", RequireUser(), ClearCart)                    // 清空购物车
		}
		
		// 订单相关API
		orders := api.Group("/orders")
		{
			orders.GET("", RequireUser(), GetOrders)                           // 获取订单列表
			orders.GET("/:id", RequireUser(), GetOrder)                        // 获取订单详情
			orders.POST("", RequireUser(), CreateOrder)                        // 创建订单
			orders.PUT("/:id/status", RequireUser(), UpdateOrderStatus)        // 更新订单状态
			orders.DELETE("/:id", RequireUser(), CancelOrder)                  // 取消订单
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