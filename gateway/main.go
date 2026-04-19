package main

import (
	"embed"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grayfalcon666/gateway/middleware"
	"github.com/grayfalcon666/gateway/router"
	"github.com/spf13/viper"
)

//go:embed swagger swagger/swagger-ui
var swaggerAssets embed.FS

func main() {
	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// Initialize JWT maker
	maker := middleware.NewSimpleJWTMaker(viper.GetString("TOKEN_SYMMETRIC_KEY"))

	// Define backend services
	backends := []router.Backend{
		// simplebank: 合并 auth/account/transfers，统一处理
		// 注意：RouterGroup 已有前缀 /api/v1，StaticRoutes 只需传后半段
		{
			Prefix:      "/api/v1/auth",
			TargetURL:   viper.GetString("SIMPLEBANK_URL"),
			StripPrefix: "/api/v1/auth",
			StaticRoutes: map[string]string{
				"/auth/register": "/v1/create_user",
				"/auth/login":    "/v1/login_user",
				"/auth/verify_email": "/v1/verify_email",
				"/auth/update":   "/v1/update_user",
			},
		},
		{
			Prefix:      "/api/v1/account",
			TargetURL:   viper.GetString("SIMPLEBANK_URL"),
			StripPrefix: "/api/v1/account",
			StaticRoutes: map[string]string{
				"/account/create": "/v1/create_account",
				"/account":        "/v1/accounts",
			},
		},
		{
			Prefix:      "/api/v1/transfers",
			TargetURL:   viper.GetString("SIMPLEBANK_URL"),
			StripPrefix: "/api/v1/transfers",
			StaticRoutes: map[string]string{
				"/transfers": "/v1/transfers",
			},
		},
		{
			Prefix:    "/api/v1/bounties",
			TargetURL: viper.GetString("ESCROW_BOUNTY_URL"),
			StaticRoutes: map[string]string{
				"/bounties":                       "/v1/bounties",
				"/bounties/:bounty_id":            "/v1/bounties/{bounty_id}",
				"/bounties/:bounty_id/accept":     "/v1/bounties/{bounty_id}/accept",
				"/bounties/:bounty_id/confirm":    "/v1/bounties/{bounty_id}/confirm",
				"/bounties/:bounty_id/complete":   "/v1/bounties/{bounty_id}/complete",
				"/bounties/:bounty_id/submit":     "/v1/bounties/{bounty_id}/submit",
				"/bounties/:bounty_id/approve":    "/v1/bounties/{bounty_id}/approve",
				"/bounties/:bounty_id/reject":     "/v1/bounties/{bounty_id}/reject",
				"/bounties/:bounty_id/cancel":     "/v1/bounties/{bounty_id}/cancel",
				"/bounties/:bounty_id/messages":   "/v1/bounties/{bounty_id}/messages",
				"/bounties/:bounty_id/comments":   "/v1/bounties/{bounty_id}/comments",
				"/bounties/:bounty_id/invitations": "/v1/bounties/{bounty_id}/invitations",
				"/bounties/applications/received": "/v1/bounties/applications/received",
			},
			MethodRoutes: map[string]string{
				"/bounties/:bounty_id:DELETE": "/v1/bounties/{bounty_id}",
			},
		},
		// Invitations
		{
			Prefix:    "/api/v1/invitations",
			TargetURL: viper.GetString("ESCROW_BOUNTY_URL"),
			StaticRoutes: map[string]string{
				"/invitations/received":           "/v1/invitations/received",
				"/invitations/sent":               "/v1/invitations/sent",
				"/invitations/:invitation_id/respond": "/v1/invitations/{invitation_id}/respond",
			},
		},
		{
			Prefix:    "/api/v1/profiles",
			TargetURL: viper.GetString("USER_PROFILE_URL"),
			StaticRoutes: map[string]string{
				"/profiles":            "/v1/profiles",
				"/profiles/:username":   "/v1/profiles/{username}",
				"/profiles/:username/role": "/v1/profiles/{username}/role",
			},
		},
		{
			Prefix:    "/api/v1/users",
			TargetURL: viper.GetString("USER_PROFILE_URL"),
			StaticRoutes: map[string]string{
				"/users/:username/reviews": "/v1/users/{username}/reviews",
				"/users/search":           "/v1/users/search",
			},
		},
		{
			Prefix:    "/api/v1/reviews",
			TargetURL: viper.GetString("USER_PROFILE_URL"),
			StaticRoutes: map[string]string{
				"/reviews": "/v1/reviews",
			},
		},
		// Hunters search (role=POSTER filter on user-profile-service)
		{
			Prefix:    "/api/v1/hunters",
			TargetURL: viper.GetString("USER_PROFILE_URL"),
			StaticRoutes: map[string]string{
				"/hunters/search": "/v1/hunters/search",
			},
		},
		{
			Prefix:    "/api/v1/payments",
			TargetURL: viper.GetString("PAYMENT_SERVICE_URL"),
			StaticRoutes: map[string]string{
				"/payments":          "/v1/payments",
				"/payments/create":   "/v1/payments/create",
			},
		},
		{
			Prefix:    "/api/v1/withdrawals",
			TargetURL: viper.GetString("PAYMENT_SERVICE_URL"),
			StaticRoutes: map[string]string{
				"/withdrawals":        "/v1/withdrawals",
				"/withdrawals/create": "/v1/withdrawals/create",
			},
		},
		// Private chat routes
		{
			Prefix:    "/api/v1/conversations",
			TargetURL: viper.GetString("ESCROW_BOUNTY_URL"),
			StaticRoutes: map[string]string{
				"/conversations":                    "/v1/conversations",
				"/conversations/unread":             "/v1/conversations/unread",
				"/conversations/:conv_id":           "/v1/conversations/{conv_id}",
				"/conversations/:conv_id/messages":  "/v1/conversations/{conv_id}/messages",
				"/conversations/:conv_id/read":      "/v1/conversations/{conv_id}/read",
			},
		},
		// Comment routes
		{
			Prefix:    "/api/v1/comments",
			TargetURL: viper.GetString("ESCROW_BOUNTY_URL"),
			StaticRoutes: map[string]string{
				"/comments/:comment_id": "/v1/comments/{comment_id}",
				"/comments":            "/v1/comments",
			},
		},
	}

	allowedOrigins := viper.GetString("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = middleware.DefaultAllowedOrigins
	}

	r := router.NewRouter(maker, allowedOrigins, backends)

	// Alipay webhook pass-through (no JWT auth, must be before swagger routes)
	alipayWebhookTarget := viper.GetString("PAYMENT_SERVICE_URL") + "/webhook/alipay"
	alipayProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL, _ = url.Parse(alipayWebhookTarget)
			req.Header.Del("Origin")
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Real-IP", req.RemoteAddr)
		},
	}
	r.Any("/webhook/alipay", func(c *gin.Context) {
		alipayProxy.ServeHTTP(c.Writer, c.Request)
	})

	// WebSocket pass-through to escrow-bounty WS server
	wsTargetBase := viper.GetString("ESCROW_BOUNTY_WS_URL")
	wsProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			target := wsTargetBase + "/ws"
			if req.URL.RawQuery != "" {
				target += "?" + req.URL.RawQuery
			}
			req.URL, _ = url.Parse(target)
			req.Header.Del("Origin")
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Real-IP", req.RemoteAddr)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("WS Proxy error: %v", err)
			http.Error(w, "WebSocket proxy error: "+err.Error(), http.StatusInternalServerError)
		},
	}
	r.GET("/ws", func(c *gin.Context) {
		wsProxy.ServeHTTP(c.Writer, c.Request)
	})

	// Swagger UI routes
	r.GET("/swagger/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// filepath starts with "/", strip it
		if filepath == "" || filepath == "/" {
			filepath = "/index.html"
		}
		file := "swagger" + filepath
		content, err := swaggerAssets.ReadFile(file)
		if err != nil {
			c.String(404, "File not found: %s", file)
			return
		}
		contentType := getContentType(file)
		c.Header("Content-Type", contentType)
		c.Data(200, contentType, content)
	})
	r.GET("/swagger-ui/*filepath", func(c *gin.Context) {
		file := "swagger/swagger-ui" + c.Param("filepath")
		content, err := swaggerAssets.ReadFile(file)
		if err != nil {
			c.String(404, "File not found: %s", file)
			return
		}
		contentType := getContentType(file)
		c.Header("Content-Type", contentType)
		c.Data(200, contentType, content)
	})

	gatewayAddr := viper.GetString("GATEWAY_PORT")
	log.Printf("🚀 API Gateway 启动中，监听 %s", gatewayAddr)
	log.Printf("📡 后端服务:")
	log.Printf("   - simplebank:           %s", viper.GetString("SIMPLEBANK_URL"))
	log.Printf("   - escrow-bounty:        %s", viper.GetString("ESCROW_BOUNTY_URL"))
	log.Printf("   - user-profile-service: %s", viper.GetString("USER_PROFILE_URL"))
	log.Printf("   - payment-service:       %s", viper.GetString("PAYMENT_SERVICE_URL"))

	if err := r.Run(gatewayAddr); err != nil {
		log.Fatalf("网关启动失败: %v", err)
	}
}

func loadConfig() error {
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("GATEWAY_PORT", "0.0.0.0:8080")
	viper.SetDefault("TOKEN_SYMMETRIC_KEY", "12345678901234567890123456789012")
	viper.SetDefault("SIMPLEBANK_URL", "http://localhost:11452")
	viper.SetDefault("ESCROW_BOUNTY_URL", "http://localhost:8087")
	viper.SetDefault("USER_PROFILE_URL", "http://localhost:8088")
	viper.SetDefault("PAYMENT_SERVICE_URL", "http://localhost:8082")
	viper.SetDefault("ESCROW_BOUNTY_WS_URL", "http://localhost:9099")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Fall back to .env file manually
			return loadEnvFile()
		}
		return err
	}
	return nil
}

func loadEnvFile() error {
	// Simple .env file loader
	data, err := os.ReadFile("app.env")
	if err != nil {
		return err
	}
	lines := splitLines(string(data))
	for _, line := range lines {
		line = trimComment(line)
		if line == "" {
			continue
		}
		parts := splitEnvLine(line)
		if len(parts) == 2 {
			viper.Set(parts[0], parts[1])
		}
	}
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			// Remove carriage return
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	return lines
}

func trimComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			inQuote = !inQuote
		}
		if line[i] == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func splitEnvLine(line string) []string {
	var key, val string
	i := 0
	for ; i < len(line) && line[i] != '='; i++ {
		if line[i] == ' ' || line[i] == '\t' {
			continue
		}
		break
	}
	key = line[:i]
	if i < len(line) {
		i++ // skip '='
	}
	// Skip whitespace
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	val = line[i:]
	// Trim surrounding quotes
	val = trimQuotes(val)
	return []string{key, val}
}

func trimQuotes(s string) string {
	s = strings.TrimLeft(s, " \t")
	s = strings.TrimRight(s, " \t")
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return s
}

func getContentType(file string) string {
	switch {
	case strings.HasSuffix(file, ".json"):
		return "application/json"
	case strings.HasSuffix(file, ".html"):
		return "text/html"
	case strings.HasSuffix(file, ".css"):
		return "text/css"
	case strings.HasSuffix(file, ".js"):
		return "application/javascript"
	case strings.HasSuffix(file, ".png"):
		return "image/png"
	case strings.HasSuffix(file, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(file, ".map"):
		return "application/json"
	default:
		return "text/plain"
	}
}

