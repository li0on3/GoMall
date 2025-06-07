package main

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 商品请求和响应结构体
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=200"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Stock       int     `json:"stock" binding:"min=0"`
	CategoryID  uint    `json:"category_id" binding:"required"`
	Images      []string `json:"images"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price,omitempty"`
	Stock       int     `json:"stock,omitempty"`
	CategoryID  uint    `json:"category_id,omitempty"`
	Images      []string `json:"images,omitempty"`
}

type ProductQueryRequest struct {
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=10"`
	CategoryID uint   `form:"category_id"`
	Keyword    string `form:"keyword"`
	MinPrice   float64 `form:"min_price"`
	MaxPrice   float64 `form:"max_price"`
	SortBy     string `form:"sort_by,default=created_at"` // created_at, price, sales_count
	SortOrder  string `form:"sort_order,default=desc"`   // asc, desc
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	ParentID    uint   `json:"parent_id,default=0"`
	SortOrder   int    `json:"sort_order,default=0"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ParentID    uint   `json:"parent_id,omitempty"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// 商品缓存管理
func CacheProduct(productID uint, product *Product) error {
	key := fmt.Sprintf("product:%d", productID)
	data, err := json.Marshal(product)
	if err != nil {
		return err
	}
	return RDB.Set(CTX, key, data, time.Hour*2).Err() // 2小时过期
}

func GetCachedProduct(productID uint) (*Product, error) {
	key := fmt.Sprintf("product:%d", productID)
	data, err := RDB.Get(CTX, key).Result()
	if err != nil {
		return nil, err
	}
	var product Product
	err = json.Unmarshal([]byte(data), &product)
	return &product, err
}

func DeleteCachedProduct(productID uint) error {
	key := fmt.Sprintf("product:%d", productID)
	return RDB.Del(CTX, key).Err()
}

// 商品列表缓存
func CacheProductList(cacheKey string, products []Product, total int64) error {
	data := map[string]interface{}{
		"products": products,
		"total":    total,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return RDB.Set(CTX, cacheKey, jsonData, time.Minute*10).Err() // 10分钟过期
}

func GetCachedProductList(cacheKey string) ([]Product, int64, error) {
	data, err := RDB.Get(CTX, cacheKey).Result()
	if err != nil {
		return nil, 0, err
	}
	
	var result map[string]interface{}
	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, 0, err
	}
	
	productsData, _ := json.Marshal(result["products"])
	var products []Product
	json.Unmarshal(productsData, &products)
	
	total := int64(result["total"].(float64))
	return products, total, nil
}

// 分类缓存管理
func CacheCategories(categories []Category) error {
	key := "categories:all"
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	return RDB.Set(CTX, key, data, time.Hour*6).Err() // 6小时过期
}

func GetCachedCategories() ([]Category, error) {
	key := "categories:all"
	data, err := RDB.Get(CTX, key).Result()
	if err != nil {
		return nil, err
	}
	var categories []Category
	err = json.Unmarshal([]byte(data), &categories)
	return categories, err
}

func DeleteCachedCategories() error {
	key := "categories:all"
	return RDB.Del(CTX, key).Err()
}

// CreateProduct 创建商品
// @Summary 创建新商品
// @Description 创建新的商品信息，包括名称、描述、价格、库存等
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "商品信息"
// @Success 200 {object} ApiResponse{data=Product} "创建成功"
// @Failure 400 {object} ApiResponse "参数验证失败"
// @Failure 404 {object} ApiResponse "商品分类不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/products [post]
func CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}

	// 检查分类是否存在
	var category Category
	if err := DB.First(&category, req.CategoryID).Error; err != nil {
		NotFoundError(c, "商品分类不存在")
		return
	}

	// 处理图片数组转JSON字符串
	imagesJSON := ""
	if len(req.Images) > 0 {
		imagesData, _ := json.Marshal(req.Images)
		imagesJSON = string(imagesData)
	}

	// 创建商品
	product := Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
		Images:      imagesJSON,
		Status:      1,
		SalesCount:  0,
	}

	if err := DB.Create(&product).Error; err != nil {
		InternalServerError(c, "商品创建失败")
		return
	}

	// 预加载分类信息
	DB.Preload("Category").First(&product, product.ID)

	// 缓存新商品
	CacheProduct(product.ID, &product)

	// 清除商品列表缓存
	pattern := "products:list:*"
	keys, _ := RDB.Keys(CTX, pattern).Result()
	if len(keys) > 0 {
		RDB.Del(CTX, keys...)
	}

	SuccessResponse(c, product)
}

// GetProducts 获取商品列表
// @Summary 获取商品列表
// @Description 获取商品列表，支持分页、分类筛选、关键字搜索、价格范围筛选和排序
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param category_id query int false "分类ID"
// @Param keyword query string false "搜索关键字"
// @Param min_price query number false "最低价格"
// @Param max_price query number false "最高价格"
// @Param sort_by query string false "排序字段" Enums(created_at, price, sales_count) default(created_at)
// @Param sort_order query string false "排序方式" Enums(asc, desc) default(desc)
// @Success 200 {object} ApiResponse{data=PaginationResponse{list=[]Product}} "查询成功"
// @Failure 400 {object} ApiResponse "参数验证失败"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Router /api/products [get]
func GetProducts(c *gin.Context) {
	var req ProductQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}

	// 参数验证
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	// 构建缓存键
	cacheKey := fmt.Sprintf("products:list:%d:%d:%d:%s:%.2f:%.2f:%s:%s",
		req.Page, req.PageSize, req.CategoryID, req.Keyword,
		req.MinPrice, req.MaxPrice, req.SortBy, req.SortOrder)

	// 尝试从缓存获取
	if products, total, err := GetCachedProductList(cacheKey); err == nil {
		PaginationSuccessResponse(c, products, total, req.Page, req.PageSize)
		return
	}

	// 构建查询
	query := DB.Model(&Product{}).Where("status = ?", 1)

	// 分类筛选
	if req.CategoryID > 0 {
		query = query.Where("category_id = ?", req.CategoryID)
	}

	// 关键字搜索
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	// 价格范围筛选
	if req.MinPrice > 0 {
		query = query.Where("price >= ?", req.MinPrice)
	}
	if req.MaxPrice > 0 {
		query = query.Where("price <= ?", req.MaxPrice)
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 排序
	sortField := req.SortBy
	if sortField != "price" && sortField != "sales_count" && sortField != "created_at" {
		sortField = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	orderBy := fmt.Sprintf("%s %s", sortField, sortOrder)

	// 分页查询
	var products []Product
	offset := (req.Page - 1) * req.PageSize
	err := query.Preload("Category").
		Order(orderBy).
		Limit(req.PageSize).
		Offset(offset).
		Find(&products).Error

	if err != nil {
		InternalServerError(c, "商品查询失败")
		return
	}

	// 缓存结果
	CacheProductList(cacheKey, products, total)

	PaginationSuccessResponse(c, products, total, req.Page, req.PageSize)
}

// GetProduct 获取商品详情
// @Summary 获取商品详情
// @Description 根据商品ID获取商品的详细信息
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} ApiResponse{data=Product} "查询成功"
// @Failure 400 {object} ApiResponse "无效的商品ID"
// @Failure 404 {object} ApiResponse "商品不存在或已下架"
// @Router /api/products/{id} [get]
func GetProduct(c *gin.Context) {
	idParam := c.Param("id")
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的商品ID")
		return
	}

	// 尝试从缓存获取
	if product, err := GetCachedProduct(uint(productID)); err == nil {
		SuccessResponse(c, product)
		return
	}

	// 从数据库查询
	var product Product
	if err := DB.Preload("Category").First(&product, productID).Error; err != nil {
		NotFoundError(c, "商品不存在")
		return
	}

	// 检查商品状态
	if product.Status != 1 {
		NotFoundError(c, "商品已下架")
		return
	}

	// 缓存商品信息
	CacheProduct(product.ID, &product)

	SuccessResponse(c, product)
}

// UpdateProduct 更新商品信息
// @Summary 更新商品信息
// @Description 更新商品的名称、描述、价格、库存、分类等信息
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Param product body UpdateProductRequest true "更新的商品信息"
// @Success 200 {object} ApiResponse{data=Product} "更新成功"
// @Failure 400 {object} ApiResponse "参数验证失败"
// @Failure 404 {object} ApiResponse "商品不存在或分类不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/products/{id} [put]
func UpdateProduct(c *gin.Context) {
	idParam := c.Param("id")
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的商品ID")
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}

	// 查询商品
	var product Product
	if err := DB.First(&product, productID).Error; err != nil {
		NotFoundError(c, "商品不存在")
		return
	}

	// 准备更新数据
	updates := make(map[string]interface{})
	
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Price > 0 {
		updates["price"] = req.Price
	}
	if req.Stock >= 0 {
		updates["stock"] = req.Stock
	}
	if req.CategoryID > 0 {
		// 检查分类是否存在
		var category Category
		if err := DB.First(&category, req.CategoryID).Error; err != nil {
			NotFoundError(c, "商品分类不存在")
			return
		}
		updates["category_id"] = req.CategoryID
	}
	if req.Images != nil {
		imagesData, _ := json.Marshal(req.Images)
		updates["images"] = string(imagesData)
	}

	// 更新商品
	if err := DB.Model(&product).Updates(updates).Error; err != nil {
		InternalServerError(c, "商品更新失败")
		return
	}

	// 重新查询更新后的商品
	DB.Preload("Category").First(&product, productID)

	// 更新缓存
	CacheProduct(product.ID, &product)

	// 清除商品列表缓存
	pattern := "products:list:*"
	keys, _ := RDB.Keys(CTX, pattern).Result()
	if len(keys) > 0 {
		RDB.Del(CTX, keys...)
	}

	SuccessResponse(c, product)
}

// DeleteProduct 删除商品（软删除）
// @Summary 删除商品
// @Description 软删除商品，将商品状态设置为已删除
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} ApiResponse{data=object{message=string}} "删除成功"
// @Failure 400 {object} ApiResponse "无效的商品ID"
// @Failure 404 {object} ApiResponse "商品不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/products/{id} [delete]
func DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的商品ID")
		return
	}

	// 查询商品
	var product Product
	if err := DB.First(&product, productID).Error; err != nil {
		NotFoundError(c, "商品不存在")
		return
	}

	// 软删除：设置状态为0
	if err := DB.Model(&product).Update("status", 0).Error; err != nil {
		InternalServerError(c, "商品删除失败")
		return
	}

	// 删除缓存
	DeleteCachedProduct(product.ID)

	// 清除商品列表缓存
	pattern := "products:list:*"
	keys, _ := RDB.Keys(CTX, pattern).Result()
	if len(keys) > 0 {
		RDB.Del(CTX, keys...)
	}

	SuccessResponse(c, gin.H{"message": "商品删除成功"})
}

// CreateCategory 创建商品分类
// @Summary 创建商品分类
// @Description 创建新的商品分类，支持创建子分类
// @Tags 商品分类
// @Accept json
// @Produce json
// @Param category body CreateCategoryRequest true "分类信息"
// @Success 200 {object} ApiResponse{data=Category} "创建成功"
// @Failure 400 {object} ApiResponse "参数验证失败"
// @Failure 404 {object} ApiResponse "父分类不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/categories [post]
func CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}

	// 检查父分类是否存在
	if req.ParentID > 0 {
		var parentCategory Category
		if err := DB.First(&parentCategory, req.ParentID).Error; err != nil {
			NotFoundError(c, "父分类不存在")
			return
		}
	}

	// 创建分类
	category := Category{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
		Status:      1,
	}

	if err := DB.Create(&category).Error; err != nil {
		InternalServerError(c, "分类创建失败")
		return
	}

	// 清除分类缓存
	DeleteCachedCategories()

	SuccessResponse(c, category)
}

// GetCategories 获取分类列表
// @Summary 获取分类列表
// @Description 获取所有启用的商品分类列表
// @Tags 商品分类
// @Accept json
// @Produce json
// @Success 200 {object} ApiResponse{data=[]Category} "查询成功"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Router /api/categories [get]
func GetCategories(c *gin.Context) {
	// 尝试从缓存获取
	if categories, err := GetCachedCategories(); err == nil {
		SuccessResponse(c, categories)
		return
	}

	// 从数据库查询
	var categories []Category
	if err := DB.Where("status = ?", 1).Order("sort_order ASC, created_at ASC").Find(&categories).Error; err != nil {
		InternalServerError(c, "分类查询失败")
		return
	}

	// 缓存分类列表
	CacheCategories(categories)

	SuccessResponse(c, categories)
}

// GetCategory 获取分类详情
// @Summary 获取分类详情
// @Description 根据分类ID获取分类的详细信息
// @Tags 商品分类
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Success 200 {object} ApiResponse{data=Category} "查询成功"
// @Failure 400 {object} ApiResponse "无效的分类ID"
// @Failure 404 {object} ApiResponse "分类不存在或已禁用"
// @Router /api/categories/{id} [get]
func GetCategory(c *gin.Context) {
	idParam := c.Param("id")
	categoryID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的分类ID")
		return
	}

	var category Category
	if err := DB.First(&category, categoryID).Error; err != nil {
		NotFoundError(c, "分类不存在")
		return
	}

	if category.Status != 1 {
		NotFoundError(c, "分类已禁用")
		return
	}

	SuccessResponse(c, category)
}

// UpdateCategory 更新分类信息
// @Summary 更新分类信息
// @Description 更新分类的名称、描述、父分类、排序等信息
// @Tags 商品分类
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Param category body UpdateCategoryRequest true "更新的分类信息"
// @Success 200 {object} ApiResponse{data=Category} "更新成功"
// @Failure 400 {object} ApiResponse "参数验证失败"
// @Failure 404 {object} ApiResponse "分类不存在或父分类不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/categories/{id} [put]
func UpdateCategory(c *gin.Context) {
	idParam := c.Param("id")
	categoryID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的分类ID")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "参数验证失败: "+err.Error())
		return
	}

	// 查询分类
	var category Category
	if err := DB.First(&category, categoryID).Error; err != nil {
		NotFoundError(c, "分类不存在")
		return
	}

	// 准备更新数据
	updates := make(map[string]interface{})
	
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ParentID >= 0 {
		// 检查父分类是否存在
		if req.ParentID > 0 {
			var parentCategory Category
			if err := DB.First(&parentCategory, req.ParentID).Error; err != nil {
				NotFoundError(c, "父分类不存在")
				return
			}
		}
		updates["parent_id"] = req.ParentID
	}
	if req.SortOrder >= 0 {
		updates["sort_order"] = req.SortOrder
	}

	// 更新分类
	if err := DB.Model(&category).Updates(updates).Error; err != nil {
		InternalServerError(c, "分类更新失败")
		return
	}

	// 重新查询更新后的分类
	DB.First(&category, categoryID)

	// 清除分类缓存
	DeleteCachedCategories()

	SuccessResponse(c, category)
}

// DeleteCategory 删除分类（软删除）
// @Summary 删除分类
// @Description 软删除分类，如果分类下有商品或子分类则不能删除
// @Tags 商品分类
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Success 200 {object} ApiResponse{data=object{message=string}} "删除成功"
// @Failure 400 {object} ApiResponse "无效的分类ID或分类下有商品/子分类"
// @Failure 404 {object} ApiResponse "分类不存在"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Security Bearer
// @Router /api/categories/{id} [delete]
func DeleteCategory(c *gin.Context) {
	idParam := c.Param("id")
	categoryID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		BadRequestError(c, "无效的分类ID")
		return
	}

	// 查询分类
	var category Category
	if err := DB.First(&category, categoryID).Error; err != nil {
		NotFoundError(c, "分类不存在")
		return
	}

	// 检查是否有商品使用此分类
	var productCount int64
	DB.Model(&Product{}).Where("category_id = ? AND status = ?", categoryID, 1).Count(&productCount)
	if productCount > 0 {
		BadRequestError(c, "该分类下有商品，无法删除")
		return
	}

	// 检查是否有子分类
	var childCount int64
	DB.Model(&Category{}).Where("parent_id = ? AND status = ?", categoryID, 1).Count(&childCount)
	if childCount > 0 {
		BadRequestError(c, "该分类下有子分类，无法删除")
		return
	}

	// 软删除：设置状态为0
	if err := DB.Model(&category).Update("status", 0).Error; err != nil {
		InternalServerError(c, "分类删除失败")
		return
	}

	// 清除分类缓存
	DeleteCachedCategories()

	SuccessResponse(c, gin.H{"message": "分类删除成功"})
}

// UploadProductImages 上传商品图片
// @Summary 上传商品图片
// @Description 上传商品图片，支持多文件上传，最多10张
// @Tags 商品管理
// @Accept multipart/form-data
// @Produce json
// @Param images formData file true "商品图片（支持jpg、jpeg、png、gif、webp格式）"
// @Success 200 {object} ApiResponse{data=object{uploaded_files=[]string,uploaded_count=int,total_files=int,errors=[]string}} "上传成功"
// @Failure 400 {object} ApiResponse "文件解析失败或文件数量超限"
// @Security Bearer
// @Router /api/products/upload [post]
func UploadProductImages(c *gin.Context) {
	// 解析多文件上传
	form, err := c.MultipartForm()
	if err != nil {
		BadRequestError(c, "文件解析失败")
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		BadRequestError(c, "请选择要上传的图片")
		return
	}

	// 检查文件数量限制
	if len(files) > 10 {
		BadRequestError(c, "最多只能上传10张图片")
		return
	}

	var uploadedFiles []string
	var uploadErrors []string

	// 确保上传目录存在
	uploadDir := AppConfig.UploadPath + "/products"
	os.MkdirAll(uploadDir, 0755)

	for _, file := range files {
		// 验证文件类型
		if !isValidImageFile(file) {
			uploadErrors = append(uploadErrors, fmt.Sprintf("文件 %s 不是有效的图片格式", file.Filename))
			continue
		}

		// 验证文件大小
		if file.Size > AppConfig.MaxFileSize {
			uploadErrors = append(uploadErrors, fmt.Sprintf("文件 %s 大小超过限制", file.Filename))
			continue
		}

		// 生成唯一文件名
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("product_%d_%s%s", time.Now().UnixNano(), generateRandomString(8), ext)
		savePath := filepath.Join(uploadDir, filename)

		// 保存文件
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("文件 %s 保存失败", file.Filename))
			continue
		}

		// 记录上传文件
		userID, _ := c.Get("user_id")
		uploadedFile := UploadedFile{
			OriginalName: file.Filename,
			FileName:     filename,
			FilePath:     "/upload/products/" + filename,
			FileSize:     file.Size,
			MimeType:     file.Header.Get("Content-Type"),
			UploadedBy:   userID.(uint),
		}

		if err := DB.Create(&uploadedFile).Error; err == nil {
			uploadedFiles = append(uploadedFiles, uploadedFile.FilePath)
		}
	}

	// 返回结果
	result := gin.H{
		"uploaded_files": uploadedFiles,
		"uploaded_count": len(uploadedFiles),
		"total_files":    len(files),
	}

	if len(uploadErrors) > 0 {
		result["errors"] = uploadErrors
	}

	SuccessResponse(c, result)
}

// 验证图片文件
func isValidImageFile(file *multipart.FileHeader) bool {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	
	// 检查MIME类型
	contentType := file.Header.Get("Content-Type")
	validTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	
	return false
}

// 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// GetHotProducts 获取热门商品
// @Summary 获取热门商品
// @Description 根据销量获取热门商品列表
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param limit query int false "返回数量限制" default(10) maximum(50)
// @Success 200 {object} ApiResponse{data=[]Product} "查询成功"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Router /api/products/hot [get]
func GetHotProducts(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// 缓存键
	cacheKey := fmt.Sprintf("products:hot:%d", limit)

	// 尝试从缓存获取
	if products, _, err := GetCachedProductList(cacheKey); err == nil {
		SuccessResponse(c, products)
		return
	}

	// 查询热门商品（按销量排序）
	var products []Product
	err := DB.Preload("Category").
		Where("status = ?", 1).
		Order("sales_count DESC, created_at DESC").
		Limit(limit).
		Find(&products).Error

	if err != nil {
		InternalServerError(c, "热门商品查询失败")
		return
	}

	// 缓存结果
	CacheProductList(cacheKey, products, int64(len(products)))

	SuccessResponse(c, products)
}

// SearchProducts 搜索商品
// @Summary 搜索商品
// @Description 根据关键字搜索商品名称和描述
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键字"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10) maximum(100)
// @Success 200 {object} ApiResponse{data=PaginationResponse{list=[]Product}} "搜索成功"
// @Failure 400 {object} ApiResponse "搜索关键字不能为空"
// @Failure 500 {object} ApiResponse "服务器内部错误"
// @Router /api/products/search [get]
func SearchProducts(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		BadRequestError(c, "搜索关键字不能为空")
		return
	}

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

	// 构建缓存键
	cacheKey := fmt.Sprintf("products:search:%s:%d:%d", keyword, page, pageSize)

	// 尝试从缓存获取
	if products, total, err := GetCachedProductList(cacheKey); err == nil {
		PaginationSuccessResponse(c, products, total, page, pageSize)
		return
	}

	// 搜索商品
	searchTerm := "%" + keyword + "%"
	query := DB.Model(&Product{}).Where("status = ? AND (name LIKE ? OR description LIKE ?)", 1, searchTerm, searchTerm)

	// 获取总数
	var total int64
	query.Count(&total)

	// 分页查询
	var products []Product
	offset := (page - 1) * pageSize
	err := query.Preload("Category").
		Order("sales_count DESC, created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&products).Error

	if err != nil {
		InternalServerError(c, "商品搜索失败")
		return
	}

	// 缓存结果
	CacheProductList(cacheKey, products, total)

	PaginationSuccessResponse(c, products, total, page, pageSize)
}