package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/service"
	"owas-api/internal/transport/http/middleware"
	"owas-api/internal/transport/http/response"
)

// StoreHandler exposes RESTful HTTP endpoints for OPPO Watch Store resources.
type StoreHandler struct {
	svc service.StoreUseCase
}

// NewStoreHandler constructs a StoreHandler with the injected StoreUseCase.
func NewStoreHandler(svc service.StoreUseCase) *StoreHandler {
	return &StoreHandler{svc: svc}
}

// SearchApps handles GET /api/v1/apps?word=...&type=0
func (h *StoreHandler) SearchApps(c *gin.Context) {
	searchType, err := parseOptionalIntQuery(c, "type", 0)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.Search(c.Request.Context(), middleware.GetDeviceProfile(c), domain.SearchParams{
		Word:       c.Query("word"),
		SearchType: searchType,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetHome handles GET /api/v1/home?start=0&size=10
func (h *StoreHandler) GetHome(c *gin.Context) {
	start, err := parseOptionalIntQuery(c, "start", 0)
	if err != nil {
		response.Error(c, err)
		return
	}
	size, err := parseOptionalIntQuery(c, "size", 10)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetHome(c.Request.Context(), middleware.GetDeviceProfile(c), domain.HomeParams{
		Start: start,
		Size:  size,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetSubject handles GET /api/v1/subjects/:subjectId
func (h *StoreHandler) GetSubject(c *gin.Context) {
	subjectID, err := parseInt64Param(c, "subjectId")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetSubject(c.Request.Context(), middleware.GetDeviceProfile(c), subjectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetAppByID handles GET /api/v1/apps/:appId?appType=1&query=0&module=home
func (h *StoreHandler) GetAppByID(c *gin.Context) {
	appID, err := parseInt64Param(c, "appId")
	if err != nil {
		response.Error(c, err)
		return
	}
	appType, err := parseOptionalIntQuery(c, "appType", 1)
	if err != nil {
		response.Error(c, err)
		return
	}
	queryFlag, err := parseOptionalIntQuery(c, "query", 0)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetAppDetailByID(c.Request.Context(), middleware.GetDeviceProfile(c), domain.AppDetailByIDParams{
		AppID:   appID,
		AppType: appType,
		Query:   queryFlag,
		Module:  c.DefaultQuery("module", "home"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetAppByPackage handles GET /api/v1/packages/:pkgName?query=0&module=action
func (h *StoreHandler) GetAppByPackage(c *gin.Context) {
	queryFlag, err := parseOptionalIntQuery(c, "query", 0)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetAppDetailByPkg(c.Request.Context(), middleware.GetDeviceProfile(c), domain.AppDetailByPkgParams{
		PkgName: c.Param("pkgName"),
		Query:   queryFlag,
		Module:  c.DefaultQuery("module", "action"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetAppDownload handles GET /api/v1/apps/:appId/versions/:verId/download?type=1&retry=2
func (h *StoreHandler) GetAppDownload(c *gin.Context) {
	appID, err := parseInt64Param(c, "appId")
	if err != nil {
		response.Error(c, err)
		return
	}
	verID, err := parseInt64Param(c, "verId")
	if err != nil {
		response.Error(c, err)
		return
	}
	retry, err := parseOptionalIntQuery(c, "retry", 2)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetDownloadInfo(c.Request.Context(), middleware.GetDeviceProfile(c), domain.DownloadInfoParams{
		AppID:        appID,
		VerID:        verID,
		DownloadType: c.DefaultQuery("type", "1"),
		Retry:        retry,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

type checkUpdatesRequest struct {
	Apps []domain.InstalledApp `json:"apps" binding:"required,min=1,dive"`
}

// CheckUpdates handles POST /api/v1/updates/check
func (h *StoreHandler) CheckUpdates(c *gin.Context) {
	var req checkUpdatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domain.NewInvalidInputError("invalid JSON request body", map[string]any{"reason": err.Error()}))
		return
	}
	res, err := h.svc.CheckUpdates(c.Request.Context(), middleware.GetDeviceProfile(c), req.Apps)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// GetProtocol handles GET /api/v1/protocols?language=zh_CN
func (h *StoreHandler) GetProtocol(c *gin.Context) {
	res, err := h.svc.GetProtocol(c.Request.Context(), middleware.GetDeviceProfile(c), c.DefaultQuery("language", "zh_CN"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// CreateOrder handles POST /api/v1/orders
func (h *StoreHandler) CreateOrder(c *gin.Context) {
	var req domain.CreateOrderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domain.NewInvalidInputError("invalid JSON request body", map[string]any{"reason": err.Error()}))
		return
	}
	res, err := h.svc.CreateOrder(c.Request.Context(), middleware.GetDeviceProfile(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res.Data)
}

// QueryOrder handles GET /api/v1/orders/:orderId?userId=...&appId=...
func (h *StoreHandler) QueryOrder(c *gin.Context) {
	appIDStr := strings.TrimSpace(c.Query("appId"))
	if appIDStr == "" {
		response.Error(c, domain.NewInvalidInputError("query parameter 'appId' is required", map[string]any{"field": "appId"}))
		return
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		response.Error(c, domain.NewInvalidInputError(
			fmt.Sprintf("invalid query parameter appId=%q: must be an integer", appIDStr),
			map[string]any{"field": "appId"},
		))
		return
	}
	res, err := h.svc.QueryOrder(c.Request.Context(), middleware.GetDeviceProfile(c), domain.QueryOrderInput{
		OrderID: c.Param("orderId"),
		UserID:  c.Query("userId"),
		AppID:   appID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res.Data)
}

// QueryPayStatus handles GET /api/v1/payments/status?appIds=31617863,37627586
func (h *StoreHandler) QueryPayStatus(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("appIds"))
	if raw == "" {
		response.Error(c, domain.NewInvalidInputError("query parameter 'appIds' is required", map[string]any{"field": "appIds"}))
		return
	}
	parts := strings.Split(raw, ",")
	appIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			response.Error(c, domain.NewInvalidInputError(
				fmt.Sprintf("invalid appId %q in 'appIds'", trimmed),
				map[string]any{"field": "appIds"},
			))
			return
		}
		appIDs = append(appIDs, id)
	}
	res, err := h.svc.QueryPayStatus(c.Request.Context(), middleware.GetDeviceProfile(c), appIDs)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res.Data)
}

func parseInt64Param(c *gin.Context, name string) (int64, error) {
	raw := strings.TrimSpace(c.Param(name))
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, domain.NewInvalidInputError(
			fmt.Sprintf("invalid path parameter %s=%q: must be an integer", name, raw),
			map[string]any{"field": name},
		)
	}
	return v, nil
}

func parseOptionalIntQuery(c *gin.Context, name string, defaultVal int) (int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return defaultVal, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, domain.NewInvalidInputError(
			fmt.Sprintf("invalid query parameter %s=%q: must be an integer", name, raw),
			map[string]any{"field": name},
		)
	}
	return v, nil
}
