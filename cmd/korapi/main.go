package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/OpenKorProject/korapi/internal/audit"
	"github.com/OpenKorProject/korapi/internal/config"
	"github.com/OpenKorProject/korapi/internal/handler"
	"github.com/OpenKorProject/korapi/internal/middleware"
	"github.com/OpenKorProject/korapi/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Config yükle
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Redis client'ını oluştur
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer rdb.Close()

	// Redis'e bağlan
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Gin router oluştur
	router := gin.Default()

	// ===== MIDDLEWARE ZINCIRI =====
	// 1. Request ID
	router.Use(middleware.RequestID())

	// 2. Logging
	router.Use(middleware.LoggingMiddleware())

	// 3. CORS + Security headers
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// 4. Rate limit (tenant/IP başına)
	rl := middleware.NewRateLimiter(rdb, cfg.RateLimitRPS)
	router.Use(rl.Middleware())

	// 5. JWT doğrulama (public paths hariç)
	jwtValidator := middleware.NewJWTValidator(cfg.AuthURL)
	publicPaths := []string{
		"/v1/auth/login",
		"/v1/auth/refresh",
		"/v1/auth/.well-known/jwks.json",
		"/healthz",
		"/readyz",
	}
	router.Use(jwtValidator.Middleware(publicPaths))

	// 6. RBAC context (JWT middleware içinde yapılıyor)

	// 7. Audit log
	auditLogger := audit.NewLogger()
	router.Use(middleware.AuditLog(auditLogger))

	// 8. Reverse proxy

	// ===== HANDLER'LAR =====
	h := handler.NewHandler()

	// Health check endpoint'leri
	router.GET("/healthz", h.Healthz)
	router.GET("/readyz", h.Readyz)

	// ===== REVERSE PROXY SETUP =====
	p := proxy.NewProxy(&http.Client{
		Timeout: 30 * time.Second,
	})

	// Upstream'ları kaydet
	p.RegisterUpstream("auth", cfg.AuthURL)
	p.RegisterUpstream("vm", cfg.VMUrl)
	p.RegisterUpstream("vol", cfg.VolURL)

	// Proxy route'ları
	// Auth
	router.GET("/v1/auth/*restOfPath", p.Handler("/v1/auth", "auth"))
	router.POST("/v1/auth/*restOfPath", p.Handler("/v1/auth", "auth"))
	router.PUT("/v1/auth/*restOfPath", p.Handler("/v1/auth", "auth"))
	router.PATCH("/v1/auth/*restOfPath", p.Handler("/v1/auth", "auth"))
	router.DELETE("/v1/auth/*restOfPath", p.Handler("/v1/auth", "auth"))

	// VM
	router.GET("/v1/vm/*restOfPath", p.Handler("/v1/vm", "vm"))
	router.POST("/v1/vm/*restOfPath", p.Handler("/v1/vm", "vm"))
	router.PUT("/v1/vm/*restOfPath", p.Handler("/v1/vm", "vm"))
	router.PATCH("/v1/vm/*restOfPath", p.Handler("/v1/vm", "vm"))
	router.DELETE("/v1/vm/*restOfPath", p.Handler("/v1/vm", "vm"))

	// Vol
	router.GET("/v1/vol/*restOfPath", p.Handler("/v1/vol", "vol"))
	router.POST("/v1/vol/*restOfPath", p.Handler("/v1/vol", "vol"))
	router.PUT("/v1/vol/*restOfPath", p.Handler("/v1/vol", "vol"))
	router.PATCH("/v1/vol/*restOfPath", p.Handler("/v1/vol", "vol"))
	router.DELETE("/v1/vol/*restOfPath", p.Handler("/v1/vol", "vol"))

	// 404 handler
	router.NoRoute(h.NotFound)

	// Server başlat
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	fmt.Printf("Gateway listening on %s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
