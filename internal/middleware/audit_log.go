package middleware

import (
	"strings"
	"time"

	"github.com/OpenKorProject/korapi/internal/audit"
	"github.com/gin-gonic/gin"
)

// AuditLog logs mutation requests after they complete.
func AuditLog(logger *audit.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if !isMutationMethod(c.Request.Method) {
			return
		}

		entry := &audit.Entry{
			RequestID: c.GetString("request_id"),
			TenantID:  c.GetString("tenant_id"),
			UserID:    c.GetString("user_id"),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			Timestamp: time.Now().UTC(),
		}

		_ = logger.Log(entry)
	}
}

// isMutationMethod reports whether the HTTP method mutates state.
func isMutationMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE"
}
