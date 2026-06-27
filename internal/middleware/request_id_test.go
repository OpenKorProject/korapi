package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_generates_id_when_absent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		if c.GetString("request_id") == "" {
			t.Error("request_id should not be empty")
		}
		c.JSON(http.StatusOK, gin.H{"request_id": c.GetString("request_id")})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get(RequestIDHeader) == "" {
		t.Errorf("expected %s response header", RequestIDHeader)
	}
}

func TestRequestID_preserves_existing_id(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	const expectedID = "custom-request-id"

	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		if c.GetString("request_id") != expectedID {
			t.Errorf("expected %s, got %s", expectedID, c.GetString("request_id"))
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, expectedID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) != expectedID {
		t.Errorf("expected %s header to equal %s", RequestIDHeader, expectedID)
	}
}
