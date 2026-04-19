package router

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grayfalcon666/gateway/middleware"
)

// Backend defines a target backend service.
type Backend struct {
	Prefix      string // e.g. "/api/v1/auth"
	TargetURL   string // e.g. "http://localhost:11452"
	StripPrefix string // e.g. "/api/v1/auth" -> "" (strips before sending)
	// Path rewrites: maps stripped path → backend path
	PathRewrite map[string]string // e.g. "/register" -> "/v1/create_user"
	// Explicit static routes for this backend: maps full gateway path → backend path
	StaticRoutes map[string]string // e.g. "/api/v1/auth/register" -> "/v1/create_user"
	// Method-specific routes: maps "METHOD /path" → backend path
	MethodRoutes map[string]string // e.g. "DELETE /api/v1/bounties/:bounty_id" -> "/v1/bounties/{bounty_id}"
}

// NewRouter creates a Gin engine with all routes configured.
func NewRouter(maker middleware.JWTMaker, allowedOrigins string, backends []Backend) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware(allowedOrigins))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"service": "api-gateway",
		})
	})

	// JWT auth middleware: allow auth endpoints and health without token
	jwtMiddleware := middleware.JWTAuthMiddleware(maker, []string{
		"/health",
		"/api/v1/auth/",
		"/api/v1/bounties",
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	v1.Use(jwtMiddleware)
	configureBackends(v1, backends)

	return r
}

func configureBackends(rg *gin.RouterGroup, backends []Backend) {
	for _, b := range backends {
		target, err := url.Parse(b.TargetURL)
		if err != nil {
			panic("invalid backend URL: " + b.TargetURL)
		}

		// Register static routes only (avoids Gin wildcard + static conflicts)
		// Build set of gwPaths that have method-specific routes so we skip Any for them
		methodGwPaths := make(map[string]bool)
		for methodRoute := range b.MethodRoutes {
			parts := strings.SplitN(methodRoute, " ", 2)
			if len(parts) == 2 {
				methodGwPaths[parts[1]] = true
			}
		}

		for gwPath, backendPath := range b.StaticRoutes {
			if methodGwPaths[gwPath] {
				continue // skip: method-specific route handles this path
			}

			pathCopy := backendPath
			proxy := httputil.NewSingleHostReverseProxy(target)
			proxy.ModifyResponse = func(resp *http.Response) error { return nil }
			proxy.Director = func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				// Get Gin path params from request header (set by handler)
				path := pathCopy
				if paramsStr := req.Header.Get("X-Gin-Params"); paramsStr != "" {
					parts := strings.Split(paramsStr, ",")
					for _, part := range parts {
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							path = strings.ReplaceAll(path, ":"+kv[0], kv[1]) // Gin :param
							path = strings.ReplaceAll(path, "{"+kv[0]+"}", kv[1]) // backend {param}
						}
					}
				}
				req.URL.Path = path
				req.Header.Del("X-Gin-Params")
				req.Header.Set("X-Forwarded-Host", req.Host)
				req.Header.Set("X-Real-IP", req.RemoteAddr)
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			}
			rg.Any(gwPath, func(c *gin.Context) {
				// Pass Gin path params to director via header
				paramsStr := ""
				for _, p := range c.Params {
					if paramsStr != "" {
						paramsStr += ","
					}
					paramsStr += p.Key + "=" + p.Value
				}
				c.Request.Header.Set("X-Gin-Params", paramsStr)
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}

		// Register method-specific routes
		for methodRoute, backendPath := range b.MethodRoutes {
			parts := strings.SplitN(methodRoute, " ", 2)
			if len(parts) != 2 {
				continue
			}
			method := parts[0]
			gwPath := parts[1]
			pathCopy := backendPath
			proxy := httputil.NewSingleHostReverseProxy(target)
			proxy.ModifyResponse = func(resp *http.Response) error { return nil }
			proxy.Director = func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				path := pathCopy
				if paramsStr := req.Header.Get("X-Gin-Params"); paramsStr != "" {
					parts := strings.Split(paramsStr, ",")
					for _, part := range parts {
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							path = strings.ReplaceAll(path, ":"+kv[0], kv[1])
							path = strings.ReplaceAll(path, "{"+kv[0]+"}", kv[1])
						}
					}
				}
				req.URL.Path = path
				req.Header.Del("X-Gin-Params")
				req.Header.Set("X-Forwarded-Host", req.Host)
				req.Header.Set("X-Real-IP", req.RemoteAddr)
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			}
			rg.Handle(method, gwPath, func(c *gin.Context) {
				paramsStr := ""
				for _, p := range c.Params {
					if paramsStr != "" {
						paramsStr += ","
					}
					paramsStr += p.Key + "=" + p.Value
				}
				c.Request.Header.Set("X-Gin-Params", paramsStr)
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		}
	}
}
