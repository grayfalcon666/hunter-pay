package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	DefaultAllowedOrigins = "http://localhost:9000,http://localhost:5173,http://localhost:8080"
)

// CORSMiddleware returns a Gin middleware that handles CORS.
// It allows credentials, methods, and headers for all configured origins.
func CORSMiddleware(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isOriginAllowed(origin, allowedOrigins string) bool {
	if origin == "" {
		return true // allow non-browser clients
	}

	// Simple comma-separated check
	// In production you'd want a more robust solution
	allowed := allowedOrigins
	for len(allowed) > 0 {
		// Find next comma
		i := 0
		for i < len(allowed) && allowed[i] != ',' {
			i++
		}
		candidate := allowed[:i]
		if len(candidate) > 0 {
			// Strip trailing whitespace
			for len(candidate) > 0 && (candidate[len(candidate)-1] == ' ' || candidate[len(candidate)-1] == ';') {
				candidate = candidate[:len(candidate)-1]
			}
			if candidate == origin {
				return true
			}
		}
		if i >= len(allowed) {
			break
		}
		allowed = allowed[i+1:]
	}
	return false
}
