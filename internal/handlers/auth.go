package handlers

import (
	"autobot/internal/database"
	"autobot/internal/middleware"
	"autobot/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginHandler 登录页面
func LoginHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "登录",
	})
}

// Login 用户登录API
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找用户
	var user models.User
	if err := database.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 设置session cookie（简化实现，实际应该使用更安全的session机制）
	c.SetCookie(middleware.SessionCookieName, strconv.Itoa(int(user.ID)), 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"user":    user.ToResponse(),
	})
}

// Register 用户注册API
func Register(c *gin.Context) {
	// 检查是否已有用户存在 - 使用事务确保数据一致性
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误，请稍后重试"})
		return
	}
	defer tx.Rollback() // 如果没有提交，自动回滚

	var userCount int64
	if err := tx.Model(&models.User{}).Count(&userCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误，请稍后重试"})
		return
	}

	if userCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "注册功能已禁用"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 在同一事务中检查用户名是否已存在
	var existingUser models.User
	if err := tx.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	// 密码长度验证
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少6位"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 在事务中创建用户
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
	}

	if err := tx.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 自动登录
	c.SetCookie(middleware.SessionCookieName, strconv.Itoa(int(user.ID)), 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"user":    user.ToResponse(),
	})
}

// Logout 用户登出API
func Logout(c *gin.Context) {
	// 清除session cookie
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// GetCurrentUser 获取当前用户信息API
func GetCurrentUser(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

// CheckRegistrationAvailable 检查是否允许注册
func CheckRegistrationAvailable(c *gin.Context) {
	var userCount int64
	if err := database.GetDB().Model(&models.User{}).Count(&userCount).Error; err != nil {
		// 数据库错误时，为安全起见，禁用注册功能
		c.JSON(http.StatusOK, gin.H{
			"registration_available": false,
			"user_count":             -1, // -1 表示无法获取用户数量
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"registration_available": userCount == 0,
		"user_count":             userCount,
	})
}
