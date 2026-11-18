package handler

import (
	"runtime"

	"github.com/gin-gonic/gin"

	"owas-api/internal/transport/http/response"
)

// HealthzHandler handles service liveness checks.
type HealthzHandler struct {
	version string
}

// NewHealthzHandler constructs a HealthzHandler with the service version label.
func NewHealthzHandler(version string) *HealthzHandler {
	if version == "" {
		version = "dev"
	}
	return &HealthzHandler{version: version}
}

// Check handles GET /healthz.
func (h *HealthzHandler) Check(c *gin.Context) {
	response.OK(c, gin.H{
		"status":    "ok",
		"service":   "owas-api",
		"version":   h.version,
		"goVersion": runtime.Version(),
	})
}
