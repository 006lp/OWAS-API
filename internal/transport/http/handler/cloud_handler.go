package handler

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/service"
	"owas-api/internal/transport/http/middleware"
	"owas-api/internal/transport/http/response"
)

// CloudHandler exposes RESTful HTTP endpoints for OPPO Health Cloud resources.
type CloudHandler struct {
	svc service.HealthUseCase
}

// NewCloudHandler constructs a CloudHandler with the injected HealthUseCase.
func NewCloudHandler(svc service.HealthUseCase) *CloudHandler {
	return &CloudHandler{svc: svc}
}

// QuerySwitchStatus handles GET /api/v1/health-cloud/switches/:switchType
func (h *CloudHandler) QuerySwitchStatus(c *gin.Context) {
	raw := strings.TrimSpace(c.Param("switchType"))
	switchType, err := strconv.Atoi(raw)
	if err != nil {
		response.Error(c, domain.NewInvalidInputError(
			fmt.Sprintf("invalid path parameter switchType=%q: must be an integer", raw),
			map[string]any{"field": "switchType"},
		))
		return
	}
	res, err := h.svc.QuerySwitchStatus(c.Request.Context(), middleware.GetDeviceProfile(c), switchType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// CheckPay handles POST /api/v1/health-cloud/pay-checks
func (h *CloudHandler) CheckPay(c *gin.Context) {
	var req domain.CheckPayInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domain.NewInvalidInputError("invalid JSON request body", map[string]any{"reason": err.Error()}))
		return
	}
	res, err := h.svc.CheckPay(c.Request.Context(), middleware.GetDeviceProfile(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// ExchangeSymmetricKey handles POST /api/v1/health-cloud/symmetric-keys/exchange
func (h *CloudHandler) ExchangeSymmetricKey(c *gin.Context) {
	var req domain.ExchangeKeyInput
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, domain.NewInvalidInputError("invalid JSON request body", map[string]any{"reason": err.Error()}))
		return
	}
	res, err := h.svc.ExchangeSymmetricKey(c.Request.Context(), middleware.GetDeviceProfile(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
