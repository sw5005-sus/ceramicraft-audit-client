package auditclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sw5005-sus/ceramicraft-audit-client/pb"
)

// auditableMethods is the set of HTTP methods that should be captured.
var auditableMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// responseWriter wraps gin.ResponseWriter to capture the status code.
type responseWriter struct {
	gin.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) WriteHeaderNow() {
	if rw.status == 0 {
		rw.status = rw.ResponseWriter.Status()
	}
	rw.ResponseWriter.WriteHeaderNow()
}

// AuditMiddleware returns a Gin middleware that records POST/PUT/PATCH/DELETE
// requests to the audit log service. It reads the userID from the Gin context
// key "userID" (set by upstream authentication middleware).
//
// serviceName is the name of the calling service (e.g. "order-service").
// auditAddr   is the gRPC address of the audit log microservice.
func AuditMiddleware(serviceName, auditAddr string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !auditableMethods[c.Request.Method] {
			c.Next()
			return
		}

		// Read and restore request body so the handler can still use it.
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Wrap the response writer to capture the status code.
		rw := &responseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = rw

		c.Next()

		// Retrieve the resolved status (gin sets it after the handler chain).
		status := c.Writer.Status()

		// Build description: METHOD URI?query | body=... | status=...
		query := ""
		if c.Request.URL.RawQuery != "" {
			query = "?" + c.Request.URL.RawQuery
		}
		description := fmt.Sprintf("%s %s%s | body=%s | status=%d",
			c.Request.Method,
			c.Request.URL.Path,
			query,
			string(bodyBytes),
			status,
		)

		// Get userID from Gin context (set by auth middleware upstream).
		var actorID int64
		if v, exists := c.Get("userID"); exists {
			switch id := v.(type) {
			case int64:
				actorID = id
			case int:
				actorID = int64(id)
			case float64:
				actorID = int64(id)
			}
		}

		client, err := GetAuditClient(auditAddr)
		if err != nil {
			log.Printf("[auditclient] failed to get audit client: %v", err)
			return
		}

		req := &pb.RecordAuditLogRequest{
			Service:     serviceName,
			ActorId:     actorID,
			Description: description,
			OccurredAt:  time.Now().UTC().Format(time.RFC3339),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.RecordAuditLog(ctx, req)
		if err != nil {
			log.Printf("[auditclient] failed to record audit log: %v", err)
		}
	}
}
