package middleware

import (
	"net/http"
	"strings"

	"pos-go/internal/models"
	"pos-go/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUser extracts the authenticated *models.User from the gin context.
func GetUser(c *gin.Context) (*models.User, bool) {
	v, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	u, ok := v.(*models.User)
	return u, ok
}

// Auth memvalidasi JWT Bearer token dan menyimpan user ke context
func Auth(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := services.ParseAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kedaluwarsa"})
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			return
		}

		user, err := authSvc.GetUserByID(userID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan"})
			return
		}

		user.PasswordHash = nil // jangan expose password hash
		c.Set("user", user)
		c.Set("user_id", userID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
