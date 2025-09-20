package middleware

import (
	"autobot/internal/database"
	"autobot/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	SessionCookieName = "autobot_session"
	UserIDKey         = "user_id"
)

// AuthMiddleware 身份鉴权中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取session cookie
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil {
			// 如果是API请求，返回401
			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
				c.Abort()
				return
			}
			// 如果是页面请求，重定向到登录页
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 验证session是否有效（这里简化处理，实际项目中应该有session存储）
		// 从cookie中解析用户ID（简化实现，实际应该使用更安全的session机制）
		userID := sessionID // 这里简化，实际应该从session存储中获取

		// 验证用户是否存在
		var user models.User
		if err := database.GetDB().First(&user, userID).Error; err != nil {
			// 清除无效cookie
			c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)

			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set(UserIDKey, user.ID)
		c.Set("user", user)
		c.Next()
	}
}

// RequireNoAuth 要求未登录的中间件（用于登录页面等）
func RequireNoAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已登录
		sessionID, err := c.Cookie(SessionCookieName)
		if err == nil && sessionID != "" {
			// 验证session是否有效
			var user models.User
			if err := database.GetDB().First(&user, sessionID).Error; err == nil {
				// 已登录，重定向到首页
				c.Redirect(http.StatusFound, "/")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// isAPIRequest 判断是否为API请求
func isAPIRequest(c *gin.Context) bool {
	path := c.Request.URL.Path
	return len(path) >= 4 && path[:4] == "/api"
}

// GetCurrentUser 获取当前登录用户
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(models.User); ok {
			return &u, true
		}
	}
	return nil, false
}

// GetCurrentUserID 获取当前登录用户ID
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	if userID, exists := c.Get(UserIDKey); exists {
		if id, ok := userID.(uint); ok {
			return id, true
		}
	}
	return 0, false
}

