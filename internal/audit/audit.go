package audit

import (
	"encoding/json"
	"fmt"
	"time"
)

// Entry represents a single audit log record.
type Entry struct {
	RequestID string    `json:"request_id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Target    string    `json:"target,omitempty"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Logger writes audit log entries.
type Logger struct{}

// NewLogger creates a new Logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Log writes an audit entry.
// TODO: write to a persistent append-only store instead of stdout
func (l *Logger) Log(e *Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
