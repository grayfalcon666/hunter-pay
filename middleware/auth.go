package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMaker interface {
	VerifyToken(token string) (*Payload, error)
}

type Payload struct {
	Username  string
	ExpiresAt int64
}

// JWTAuthMiddleware returns a Gin middleware that validates JWT tokens.
// It allows requests matching any of the given allowedPaths without authentication.
// The Authorization header is forwarded as grpc-metadata "authorization" to backend.
func JWTAuthMiddleware(maker JWTMaker, allowedPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check whitelist
		path := c.Request.URL.Path
		for _, allowed := range allowedPaths {
			if strings.HasPrefix(path, allowed) {
				c.Next()
				return
			}
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供 Authorization 凭证"})
			return
		}

		// Parse Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式无效"})
			return
		}

		tokenString := parts[1]

		// Verify token
		payload, err := maker.VerifyToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token 无效或已过期"})
			return
		}

		// Store username in context for downstream use
		c.Set("username", payload.Username)
		c.Set("token", tokenString)

		c.Next()
	}
}

// VerifyToken implements the JWTMaker interface using jwt-go library.
type SimpleJWTMaker struct {
	secretKey string
}

func NewSimpleJWTMaker(secretKey string) *SimpleJWTMaker {
	return &SimpleJWTMaker{secretKey: secretKey}
}

func (m *SimpleJWTMaker) VerifyToken(tokenString string) (*Payload, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(m.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	username, ok := claims["username"].(string)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	var expiresAt int64
	switch exp := claims["exp"].(type) {
	case float64:
		expiresAt = int64(exp)
	case int64:
		expiresAt = exp
	}

	return &Payload{Username: username, ExpiresAt: expiresAt}, nil
}
