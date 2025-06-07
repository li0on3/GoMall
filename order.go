package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 订单状态常量
const (
	OrderStatusPending   = "pending"   // 待支付
	OrderStatusPaid      = "paid"      // 已支付
	OrderStatusShipped   = "shipped"   // 已发货
	OrderStatusDelivered = "delivered" // 已送达
	OrderStatusCancelled = "cancelled" // 已取消
)

// 购物车相关请求结构
type AddCartRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,min=1"`
}

type UpdateCartRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// 订单相关请求结构
type CreateOrderRequest struct {
	ShippingAddress string `json:"shipping_address" binding:"required"`
	CartItemIDs     []uint `json:"cart_item_ids" binding:"required"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// 订单处理任务
type OrderJob struct {
	OrderID uint
	UserID  uint
	Type    string // "create", "update", "cancel"
	Data    interface{}
	Result  chan error
}

// 库存管理器
type StockManager struct {
	stocks map[uint]*StockChannel
	mutex  sync.RWMutex
}

type StockChannel struct {
	productID uint
	ch        chan int
	current   int
}

var (
	// 全局库存管理器
	GlobalStockManager *StockManager
	// 订单处理通道
	OrderJobQueue chan OrderJob
	// 工作协程数量
	WorkerCount = 5
)

// 初始化订单服务
func InitOrderService() {
	GlobalStockManager = &StockManager{
		stocks: make(map[uint]*StockChannel),
	}
	
	// 创建订单任务队列
	OrderJobQueue = make(chan OrderJob, 100)
	
	// 启动工作协程
	for i := 0; i < WorkerCount; i++ {
		go OrderWorker(i)
	}
	
	log.Printf("订单服务初始化完成，启动了 %d 个工作协程", WorkerCount)
}

// 订单处理工作协程
func OrderWorker(workerID int) {
	log.Printf("订单工作协程 %d 启动", workerID)
	
	for job := range OrderJobQueue {
		var err error
		start := time.Now()
		
		switch job.Type {
		case "create":
			err = processCreateOrder(job)
		case "update":
			err = processUpdateOrder(job)
		case "cancel":
			err = processCancelOrder(job)
		default:
			err = fmt.Errorf("未知的订单处理类型: %s", job.Type)
		}
		
		duration := time.Since(start)
		log.Printf("工作协程 %d 处理订单任务 %s (ID:%d) 耗时: %v", 
			workerID, job.Type, job.OrderID, duration)
		
		// 返回结果
		if job.Result != nil {
			job.Result <- err
		}
	}
}

// 处理创建订单任务
func processCreateOrder(job OrderJob) error {
	orderData := job.Data.(CreateOrderRequest)
	
	// 并发检查库存和创建订单
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	
	// 协程1: 检查库存
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := checkInventoryForOrder(job.UserID, orderData.CartItemIDs); err != nil {
			errors <- fmt.Errorf("库存检查失败: %v", err)
		}
	}()
	
	// 协程2: 计算订单金额
	wg.Add(1)
	var totalAmount float64
	go func() {
		defer wg.Done()
		if amount, err := calculateOrderAmount(orderData.CartItemIDs); err != nil {
			errors <- fmt.Errorf("金额计算失败: %v", err)
		} else {
			totalAmount = amount
		}
	}()
	
	wg.Wait()
	close(errors)
	
	// 检查是否有错误
	for err := range errors {
		if err != nil {
			return err
		}
	}
	
	// 开始创建订单（数据库事务）
	return createOrderInDB(job.UserID, orderData, totalAmount)
}

// 处理更新订单任务
func processUpdateOrder(job OrderJob) error {
	updateData := job.Data.(UpdateOrderStatusRequest)
	
	var order Order
	if err := DB.First(&order, job.OrderID).Error; err != nil {
		return fmt.Errorf("订单不存在")
	}
	
	// 更新订单状态
	if err := DB.Model(&order).Update("status", updateData.Status).Error; err != nil {
		return fmt.Errorf("订单状态更新失败: %v", err)
	}
	
	// 如果是取消订单，需要恢复库存
	if updateData.Status == OrderStatusCancelled {
		go restoreOrderStock(job.OrderID)
	}
	
	return nil
}

// 处理取消订单任务
func processCancelOrder(job OrderJob) error {
	return processUpdateOrder(OrderJob{
		OrderID: job.OrderID,
		UserID:  job.UserID,
		Type:    "update",
		Data:    UpdateOrderStatusRequest{Status: OrderStatusCancelled},
	})
}

// 库存管理相关函数

// 获取或创建库存通道
func (sm *StockManager) getStockChannel(productID uint) *StockChannel {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	if stockCh, exists := sm.stocks[productID]; exists {
		return stockCh
	}
	
	// 从数据库获取当前库存
	var product Product
	if err := DB.First(&product, productID).Error; err != nil {
		return nil
	}
	
	stockCh := &StockChannel{
		productID: productID,
		ch:        make(chan int, 1),
		current:   product.Stock,
	}
	
	// 初始化通道
	stockCh.ch <- product.Stock
	sm.stocks[productID] = stockCh
	
	return stockCh
}

// 扣减库存
func (sm *StockManager) DeductStock(productID uint, quantity int) error {
	stockCh := sm.getStockChannel(productID)
	if stockCh == nil {
		return fmt.Errorf("无法获取商品 %d 的库存信息", productID)
	}
	
	// 从通道获取当前库存
	currentStock := <-stockCh.ch
	
	if currentStock < quantity {
		// 库存不足，放回原值
		stockCh.ch <- currentStock
		return fmt.Errorf("商品 %d 库存不足，当前库存: %d，需要: %d", productID, currentStock, quantity)
	}
	
	newStock := currentStock - quantity
	stockCh.ch <- newStock
	stockCh.current = newStock
	
	// 同步更新数据库
	go func() {
		DB.Model(&Product{}).Where("id = ?", productID).Update("stock", newStock)
	}()
	
	return nil
}

// 恢复库存
func (sm *StockManager) RestoreStock(productID uint, quantity int) error {
	stockCh := sm.getStockChannel(productID)
	if stockCh == nil {
		return fmt.Errorf("无法获取商品 %d 的库存信息", productID)
	}
	
	currentStock := <-stockCh.ch
	newStock := currentStock + quantity
	stockCh.ch <- newStock
	stockCh.current = newStock
	
	// 同步更新数据库
	go func() {
		DB.Model(&Product{}).Where("id = ?", productID).Update("stock", newStock)
	}()
	
	return nil
}

// 购物车功能实现

// 添加商品到购物车
func AddToCart(c *gin.Context) {
	var req AddCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 检查商品是否存在
	var product Product
	if err := DB.First(&product, req.ProductID).Error; err != nil {
		NotFoundError(c, "商品不存在")
		return
	}
	
	// 检查商品状态
	if product.Status != 1 {
		BadRequestError(c, "商品已下架")
		return
	}
	
	// 检查库存
	if product.Stock < req.Quantity {
		BadRequestError(c, fmt.Sprintf("库存不足，当前库存: %d", product.Stock))
		return
	}
	
	// 查看购物车中是否已有该商品
	var existingItem CartItem
	result := DB.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&existingItem)
	
	if result.Error == nil {
		// 更新数量
		newQuantity := existingItem.Quantity + req.Quantity
		if product.Stock < newQuantity {
			BadRequestError(c, "库存不足")
			return
		}
		
		if err := DB.Model(&existingItem).Update("quantity", newQuantity).Error; err != nil {
			InternalServerError(c, "购物车更新失败")
			return
		}
		
		// 预加载商品信息
		DB.Preload("Product").First(&existingItem, existingItem.ID)
		SuccessResponse(c, existingItem)
	} else {
		// 创建新的购物车项
		cartItem := CartItem{
			UserID:    userID.(uint),
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
		}
		
		if err := DB.Create(&cartItem).Error; err != nil {
			InternalServerError(c, "添加到购物车失败")
			return
		}
		
		// 预加载商品信息
		DB.Preload("Product").First(&cartItem, cartItem.ID)
		SuccessResponse(c, cartItem)
	}
}

// 获取购物车列表
func GetCart(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var cartItems []CartItem
	if err := DB.Preload("Product").Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		InternalServerError(c, "购物车查询失败")
		return
	}
	
	// 计算总金额
	var totalAmount float64
	for _, item := range cartItems {
		totalAmount += item.Product.Price * float64(item.Quantity)
	}
	
	result := gin.H{
		"items":        cartItems,
		"total_amount": totalAmount,
		"total_count":  len(cartItems),
	}
	
	SuccessResponse(c, result)
}

// 更新购物车项
func UpdateCartItem(c *gin.Context) {
	cartItemID := c.Param("id")
	itemID, err := strconv.ParseUint(cartItemID, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的购物车项ID")
		return
	}
	
	var req UpdateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 查询购物车项
	var cartItem CartItem
	if err := DB.Where("id = ? AND user_id = ?", itemID, userID).First(&cartItem).Error; err != nil {
		NotFoundError(c, "购物车项不存在")
		return
	}
	
	// 检查库存
	var product Product
	if err := DB.First(&product, cartItem.ProductID).Error; err != nil {
		InternalServerError(c, "商品查询失败")
		return
	}
	
	if product.Stock < req.Quantity {
		BadRequestError(c, fmt.Sprintf("库存不足，当前库存: %d", product.Stock))
		return
	}
	
	// 更新数量
	if err := DB.Model(&cartItem).Update("quantity", req.Quantity).Error; err != nil {
		InternalServerError(c, "购物车更新失败")
		return
	}
	
	// 预加载商品信息
	DB.Preload("Product").First(&cartItem, cartItem.ID)
	SuccessResponse(c, cartItem)
}

// 删除购物车项
func DeleteCartItem(c *gin.Context) {
	cartItemID := c.Param("id")
	itemID, err := strconv.ParseUint(cartItemID, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的购物车项ID")
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 删除购物车项
	result := DB.Where("id = ? AND user_id = ?", itemID, userID).Delete(&CartItem{})
	if result.Error != nil {
		InternalServerError(c, "删除失败")
		return
	}
	
	if result.RowsAffected == 0 {
		NotFoundError(c, "购物车项不存在")
		return
	}
	
	SuccessResponse(c, gin.H{"message": "删除成功"})
}

// 清空购物车
func ClearCart(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	if err := DB.Where("user_id = ?", userID).Delete(&CartItem{}).Error; err != nil {
		InternalServerError(c, "清空购物车失败")
		return
	}
	
	SuccessResponse(c, gin.H{"message": "购物车已清空"})
}

// 订单功能实现

// 创建订单（使用并发处理）
func CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 创建订单任务
	orderJob := OrderJob{
		UserID: userID.(uint),
		Type:   "create",
		Data:   req,
		Result: make(chan error, 1),
	}
	
	// 提交到协程池处理
	OrderJobQueue <- orderJob
	
	// 等待处理结果
	select {
	case err := <-orderJob.Result:
		if err != nil {
			InternalServerError(c, "订单创建失败: "+err.Error())
			return
		}
		
		// 获取创建的订单信息
		var order Order
		DB.Preload("OrderItems.Product").Where("user_id = ?", userID).Order("created_at DESC").First(&order)
		
		SuccessResponse(c, order)
		
	case <-time.After(30 * time.Second):
		InternalServerError(c, "订单创建超时")
		return
	}
}

// 获取订单列表
func GetOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	page := 1
	pageSize := 10
	
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	
	if sizeParam := c.Query("page_size"); sizeParam != "" {
		if s, err := strconv.Atoi(sizeParam); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}
	
	// 查询订单
	var orders []Order
	var total int64
	
	query := DB.Model(&Order{}).Where("user_id = ?", userID)
	query.Count(&total)
	
	offset := (page - 1) * pageSize
	err := query.Preload("OrderItems.Product").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&orders).Error
	
	if err != nil {
		InternalServerError(c, "订单查询失败")
		return
	}
	
	PaginationSuccessResponse(c, orders, total, page, pageSize)
}

// 获取订单详情
func GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	oID, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的订单ID")
		return
	}
	
	userID, _ := c.Get("user_id")
	
	var order Order
	if err := DB.Preload("OrderItems.Product").
		Where("id = ? AND user_id = ?", oID, userID).
		First(&order).Error; err != nil {
		NotFoundError(c, "订单不存在")
		return
	}
	
	SuccessResponse(c, order)
}

// 更新订单状态
func UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	oID, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的订单ID")
		return
	}
	
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}
	
	// 验证状态值
	validStatuses := map[string]bool{
		OrderStatusPending:   true,
		OrderStatusPaid:      true,
		OrderStatusShipped:   true,
		OrderStatusDelivered: true,
		OrderStatusCancelled: true,
	}
	
	if !validStatuses[req.Status] {
		BadRequestError(c, "无效的订单状态")
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 创建更新任务
	updateJob := OrderJob{
		OrderID: uint(oID),
		UserID:  userID.(uint),
		Type:    "update",
		Data:    req,
		Result:  make(chan error, 1),
	}
	
	// 提交到协程池处理
	OrderJobQueue <- updateJob
	
	// 等待处理结果
	select {
	case err := <-updateJob.Result:
		if err != nil {
			InternalServerError(c, "订单状态更新失败: "+err.Error())
			return
		}
		
		SuccessResponse(c, gin.H{"message": "订单状态更新成功"})
		
	case <-time.After(10 * time.Second):
		InternalServerError(c, "订单状态更新超时")
		return
	}
}

// 取消订单
func CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	oID, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的订单ID")
		return
	}
	
	userID, _ := c.Get("user_id")
	
	// 检查订单状态
	var order Order
	if err := DB.Where("id = ? AND user_id = ?", oID, userID).First(&order).Error; err != nil {
		NotFoundError(c, "订单不存在")
		return
	}
	
	// 只有待支付的订单可以取消
	if order.Status != OrderStatusPending {
		BadRequestError(c, "只有待支付的订单可以取消")
		return
	}
	
	// 创建取消任务
	cancelJob := OrderJob{
		OrderID: uint(oID),
		UserID:  userID.(uint),
		Type:    "cancel",
		Result:  make(chan error, 1),
	}
	
	// 提交到协程池处理
	OrderJobQueue <- cancelJob
	
	// 等待处理结果
	select {
	case err := <-cancelJob.Result:
		if err != nil {
			InternalServerError(c, "订单取消失败: "+err.Error())
			return
		}
		
		SuccessResponse(c, gin.H{"message": "订单取消成功"})
		
	case <-time.After(10 * time.Second):
		InternalServerError(c, "订单取消超时")
		return
	}
}

// 辅助函数

// 检查订单库存
func checkInventoryForOrder(userID uint, cartItemIDs []uint) error {
	for _, itemID := range cartItemIDs {
		var cartItem CartItem
		if err := DB.Preload("Product").Where("id = ? AND user_id = ?", itemID, userID).First(&cartItem).Error; err != nil {
			return fmt.Errorf("购物车项 %d 不存在", itemID)
		}
		
		// 并发检查库存
		if err := GlobalStockManager.DeductStock(cartItem.ProductID, cartItem.Quantity); err != nil {
			return err
		}
	}
	
	return nil
}

// 计算订单金额
func calculateOrderAmount(cartItemIDs []uint) (float64, error) {
	var totalAmount float64
	
	for _, itemID := range cartItemIDs {
		var cartItem CartItem
		if err := DB.Preload("Product").First(&cartItem, itemID).Error; err != nil {
			return 0, fmt.Errorf("购物车项 %d 不存在", itemID)
		}
		
		totalAmount += cartItem.Product.Price * float64(cartItem.Quantity)
	}
	
	return totalAmount, nil
}

// 在数据库中创建订单
func createOrderInDB(userID uint, req CreateOrderRequest, totalAmount float64) error {
	// 开始数据库事务
	tx := DB.Begin()
	
	// 生成订单号
	orderNo := generateOrderNumber()
	
	// 创建订单
	order := Order{
		UserID:          userID,
		OrderNo:         orderNo,
		TotalAmount:     totalAmount,
		Status:          OrderStatusPending,
		ShippingAddress: req.ShippingAddress,
	}
	
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("订单创建失败: %v", err)
	}
	
	// 创建订单项并清除购物车
	for _, itemID := range req.CartItemIDs {
		var cartItem CartItem
		if err := tx.Preload("Product").Where("id = ? AND user_id = ?", itemID, userID).First(&cartItem).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("购物车项 %d 不存在", itemID)
		}
		
		// 创建订单项
		orderItem := OrderItem{
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
			Price:     cartItem.Product.Price,
		}
		
		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("订单项创建失败: %v", err)
		}
		
		// 删除购物车项
		if err := tx.Delete(&cartItem).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("购物车清理失败: %v", err)
		}
		
		// 更新商品销量
		tx.Model(&Product{}).Where("id = ?", cartItem.ProductID).
			UpdateColumn("sales_count", gorm.Expr("sales_count + ?", cartItem.Quantity))
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("事务提交失败: %v", err)
	}
	
	return nil
}

// 恢复订单库存
func restoreOrderStock(orderID uint) {
	var orderItems []OrderItem
	if err := DB.Where("order_id = ?", orderID).Find(&orderItems).Error; err != nil {
		log.Printf("查询订单项失败: %v", err)
		return
	}
	
	for _, item := range orderItems {
		if err := GlobalStockManager.RestoreStock(item.ProductID, item.Quantity); err != nil {
			log.Printf("恢复库存失败 - 商品ID: %d, 数量: %d, 错误: %v", 
				item.ProductID, item.Quantity, err)
		}
	}
}

// 生成订单号
func generateOrderNumber() string {
	timestamp := time.Now().Unix()
	random := rand.Intn(9999)
	return fmt.Sprintf("OM%d%04d", timestamp, random)
}