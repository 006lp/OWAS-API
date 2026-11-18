package httptransport

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/service"
	"owas-api/internal/transport/http/handler"
	"owas-api/internal/transport/http/middleware"
	"owas-api/internal/transport/http/response"
)

// RouterDeps aggregates dependencies required to construct the Gin HTTP engine.
type RouterDeps struct {
	StoreService   service.StoreUseCase
	HealthService  service.HealthUseCase
	DefaultProfile domain.DeviceProfile
	Logger         *slog.Logger
	Version        string
	GinMode        string
}

// NewRouter creates and configures the Gin engine with all RESTful routes and middleware.
func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.GinMode != "" {
		gin.SetMode(deps.GinMode)
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(
		gin.Recovery(),
		middleware.RequestID(),
		middleware.Logger(deps.Logger),
	)

	r.NoRoute(func(c *gin.Context) {
		response.CustomError(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "endpoint not found")
	})
	r.NoMethod(func(c *gin.Context) {
		response.CustomError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method not allowed")
	})

	healthz := handler.NewHealthzHandler(deps.Version)
	r.GET("/healthz", healthz.Check)

	storeHandler := handler.NewStoreHandler(deps.StoreService)
	cloudHandler := handler.NewCloudHandler(deps.HealthService)

	v1 := r.Group("/api/v1", middleware.DeviceProfile(deps.DefaultProfile))
	{
		// Watch Store resources (11 upstream APIs)
		v1.GET("/apps", storeHandler.SearchApps)
		v1.GET("/apps/:appId", storeHandler.GetAppByID)
		v1.GET("/apps/:appId/versions/:verId/download", storeHandler.GetAppDownload)
		v1.GET("/packages/:pkgName", storeHandler.GetAppByPackage)
		v1.POST("/updates/check", storeHandler.CheckUpdates)
		v1.GET("/home", storeHandler.GetHome)
		v1.GET("/subjects/:subjectId", storeHandler.GetSubject)
		v1.GET("/protocols", storeHandler.GetProtocol)
		v1.POST("/orders", storeHandler.CreateOrder)
		v1.GET("/orders/:orderId", storeHandler.QueryOrder)
		v1.GET("/payments/status", storeHandler.QueryPayStatus)

		// Health Cloud resources (3 upstream APIs)
		cloud := v1.Group("/health-cloud")
		{
			cloud.GET("/switches/:switchType", cloudHandler.QuerySwitchStatus)
			cloud.POST("/pay-checks", cloudHandler.CheckPay)
			cloud.POST("/symmetric-keys/exchange", cloudHandler.ExchangeSymmetricKey)
		}
	}

	return r
}
