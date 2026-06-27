package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenKorProject/korapi/internal/config"
	"github.com/OpenKorProject/korapi/internal/handler"
	"github.com/gin-gonic/gin"
)

// TestHealthz healthz endpoint'i test et.
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

	if !contains(w.Body.String(), "ok") {
		t.Errorf("Expected 'ok' in response, got %s", w.Body.String())
	}
}

// TestReadyz readyz endpoint'i test et.
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

	if !contains(w.Body.String(), "ready") {
		t.Errorf("Expected 'ready' in response, got %s", w.Body.String())
	}
}

// TestNotFound 404 handler'ı test et.
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

	if !contains(w.Body.String(), "NOT_FOUND") {
		t.Errorf("Expected 'NOT_FOUND' in response, got %s", w.Body.String())
	}
}

// TestConfigLoad config yükleme test et.
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

// contains string'in içerip içermediğini kontrol et.
func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && (haystack == needle ||
		fmt.Sprintf("%v", haystack) != "" && fmt.Sprintf("%v", needle) != "")
}