package middleware

import (
	"net/http"
	"strings"

	"investment-tracker/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])
		c.Set("user_role", claims["role"])
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func ViewOnlyCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		viewAsUser := c.GetHeader("X-View-As-User")
		if viewAsUser == "" {
			c.Next()
			return
		}

		// Only admins can impersonate
		role, _ := c.Get("user_role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view as other users"})
			c.Abort()
			return
		}

		// Block non-GET requests when impersonating
		if c.Request.Method != http.MethodGet {
			c.JSON(http.StatusForbidden, gin.H{"error": "View-only mode: modifications not allowed"})
			c.Abort()
			return
		}

		c.Set("view_user_id", viewAsUser)
		c.Next()
	}
}

// GetEffectiveUserID returns the user ID to use for data queries.
// If admin is impersonating, returns the impersonated user's ID.
func GetEffectiveUserID(c *gin.Context) string {
	if viewUserID, exists := c.Get("view_user_id"); exists {
		return viewUserID.(string)
	}
	userID, _ := c.Get("user_id")
	return userID.(string)
}
