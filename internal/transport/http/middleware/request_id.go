package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"owas-api/internal/protocol/crypto"
	"owas-api/internal/transport/http/response"
)

const headerRequestID = "X-Request-ID"

// RequestID ensures every request has a correlation ID in gin.Context and response headers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader(headerRequestID))
		if reqID == "" || len(reqID) > 128 {
			if generated, err := crypto.GenerateUUIDHex(); err == nil {
				reqID = generated
			}
		}
		c.Set(response.ContextKeyRequestID, reqID)
		c.Header(headerRequestID, reqID)
		c.Next()
	}
}
