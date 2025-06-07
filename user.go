package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// JWTClaims JWT声明结构体
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.StandardClaims
}

// 请求和响应结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Phone    string `json:"phone"`
	RealName string `json:"real_name"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateProfileRequest struct {
	Phone    string `json:"phone"`
	RealName string `json:"real_name"`
	Avatar   string `json:"avatar"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// MD5加密密码
func HashPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	return HashPassword(password) == hashedPassword
}

// 生成JWT Token
func GenerateJWT(user *User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24 * 7).Unix(), // 7天过期
			IssuedAt:  time.Now().Unix(),
			Issuer:    "gomall",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(AppConfig.JWTSecret))
}

// 解析JWT Token
func ParseJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的token")
}

// JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			ErrorResponse(c, http.StatusUnauthorized, "缺少认证token")
			c.Abort()
			return
		}

		// 处理Bearer token格式
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := ParseJWT(token)
		if err != nil {
			ErrorResponse(c, http.StatusUnauthorized, "无效的token")
			c.Abort()
			return
		}

		// 检查token是否过期
		if claims.ExpiresAt < time.Now().Unix() {
			ErrorResponse(c, http.StatusUnauthorized, "token已过期")
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// 用户注册
func UserRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	// 检查用户名是否已存在
	var existingUser User
	if err := DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		ErrorResponse(c, http.StatusConflict, "用户名已存在")
		return
	}

	// 检查邮箱是否已存在
	if err := DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		ErrorResponse(c, http.StatusConflict, "邮箱已存在")
		return
	}

	// 创建新用户
	user := User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: HashPassword(req.Password),
		Phone:        req.Phone,
		RealName:     req.RealName,
		Status:       1,
	}

	if err := DB.Create(&user).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "用户创建失败")
		return
	}

	// 生成JWT token
	token, err := GenerateJWT(&user)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "token生成失败")
		return
	}

	// 缓存用户会话到Redis
	CacheUserSession(user.ID, token)

	SuccessResponse(c, LoginResponse{
		Token: token,
		User:  user,
	})
}

// 用户登录
func UserLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	// 查找用户（支持用户名或邮箱登录）
	var user User
	if err := DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "用户不存在或密码错误")
		return
	}

	// 验证密码
	if !VerifyPassword(req.Password, user.PasswordHash) {
		ErrorResponse(c, http.StatusUnauthorized, "用户不存在或密码错误")
		return
	}

	// 检查用户状态
	if user.Status != 1 {
		ErrorResponse(c, http.StatusForbidden, "用户账号已被禁用")
		return
	}

	// 生成JWT token
	token, err := GenerateJWT(&user)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "token生成失败")
		return
	}

	// 缓存用户会话到Redis
	CacheUserSession(user.ID, token)

	SuccessResponse(c, LoginResponse{
		Token: token,
		User:  user,
	})
}

// 获取用户信息
func GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "用户不存在")
		return
	}

	SuccessResponse(c, user)
}

// 更新用户信息
func UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 更新用户信息
	updates := map[string]interface{}{}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.RealName != "" {
		updates["real_name"] = req.RealName
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}

	if err := DB.Model(&user).Updates(updates).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "用户信息更新失败")
		return
	}

	// 重新查询更新后的用户信息
	DB.First(&user, userID)

	SuccessResponse(c, user)
}

// 修改密码
func ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 验证旧密码
	if !VerifyPassword(req.OldPassword, user.PasswordHash) {
		ErrorResponse(c, http.StatusBadRequest, "原密码错误")
		return
	}

	// 更新密码
	newPasswordHash := HashPassword(req.NewPassword)
	if err := DB.Model(&user).Update("password_hash", newPasswordHash).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "密码更新失败")
		return
	}

	SuccessResponse(c, gin.H{"message": "密码修改成功"})
}

// 用户会话缓存管理
func CacheUserSession(userID uint, token string) error {
	key := fmt.Sprintf("session:user:%d", userID)
	return RDB.Set(CTX, key, token, time.Hour*24*7).Err() // 7天过期
}

// 获取用户会话缓存
func GetUserSession(userID uint) (string, error) {
	key := fmt.Sprintf("session:user:%d", userID)
	return RDB.Get(CTX, key).Result()
}

// 删除用户会话缓存
func DeleteUserSession(userID uint) error {
	key := fmt.Sprintf("session:user:%d", userID)
	return RDB.Del(CTX, key).Err()
}

// 用户登出
func UserLogout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 删除Redis中的会话缓存
	if uid, ok := userID.(uint); ok {
		DeleteUserSession(uid)
	}

	SuccessResponse(c, gin.H{"message": "退出登录成功"})
}

// 根据ID获取用户信息（内部使用）
func GetUserByID(userID uint) (*User, error) {
	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// 检查用户权限中间件
func RequireUser() gin.HandlerFunc {
	return JWTAuthMiddleware()
}

// 管理员权限中间件（如果需要）
func RequireAdmin() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		JWTAuthMiddleware()(c)
		if c.IsAborted() {
			return
		}

		userID, _ := c.Get("user_id")
		user, err := GetUserByID(userID.(uint))
		if err != nil {
			ErrorResponse(c, http.StatusUnauthorized, "用户不存在")
			c.Abort()
			return
		}

		// 这里可以检查用户角色，目前暂时使用用户ID=1作为管理员
		if user.ID != 1 {
			ErrorResponse(c, http.StatusForbidden, "需要管理员权限")
			c.Abort()
			return
		}

		c.Next()
	})
}