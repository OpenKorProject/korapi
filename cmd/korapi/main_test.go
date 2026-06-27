package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenKorProject/korapi/internal/config"
	"github.com/OpenKorProject/korapi/internal/handler"
	"github.com/gin-gonic/gin"
)

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	h := handler.NewHandler()
	router.GET("/healthz", h.Healthz)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/healthz", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ok") {
		t.Errorf("Expected 'ok' in response, got %s", w.Body.String())
	}
}

func TestReadyz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	h := handler.NewHandler()
	router.GET("/readyz", h.Readyz)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/readyz", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ready") {
		t.Errorf("Expected 'ready' in response, got %s", w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	h := handler.NewHandler()
	router.NoRoute(h.NotFound)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "NOT_FOUND") {
		t.Errorf("Expected 'NOT_FOUND' in response, got %s", w.Body.String())
	}
}

func TestConfigLoad(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.Port == 0 {
		t.Errorf("Port should not be 0")
	}
	if cfg.AuthURL == "" {
		t.Errorf("AuthURL should not be empty")
	}
	if cfg.RedisAddr == "" {
		t.Errorf("RedisAddr should not be empty")
	}
}
