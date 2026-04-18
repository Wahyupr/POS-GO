package middleware

import (
	"net/http"

	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// RequireRole hanya mengizinkan request dari role yang ditentukan
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[string(r)] = true
	}

	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}

		if !allowed[role.(string)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			return
		}

		c.Next()
	}
}
