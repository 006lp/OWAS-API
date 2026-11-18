package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/transport/http/response"
)

const contextKeyDeviceProfile = "device_profile"

// DeviceProfile extracts optional OPPO Watch device profile overrides from HTTP headers.
func DeviceProfile(base domain.DeviceProfile) gin.HandlerFunc {
	def := base.WithDefaults()
	return func(c *gin.Context) {
		p := def
		overrideString(&p.Brand, c.GetHeader("X-Watch-Brand"))
		overrideString(&p.Product, c.GetHeader("X-Watch-Product"))
		overrideString(&p.Model, c.GetHeader("X-Watch-Model"))
		overrideString(&p.AndroidRelease, c.GetHeader("X-Watch-Android-Release"))
		overrideString(&p.HeyRelease, c.GetHeader("X-Watch-Hey-Release"))
		overrideString(&p.ColorOSVer, c.GetHeader("X-Watch-ColorOS-Ver"))
		overrideString(&p.SysVer, c.GetHeader("X-Watch-Sys-Ver"))
		overrideString(&p.Locale, c.GetHeader("X-Watch-Locale"))
		overrideString(&p.Support, c.GetHeader("X-Watch-Support"))
		overrideString(&p.Linkage, c.GetHeader("X-Watch-Linkage"))
		overrideString(&p.InstantSwClose, c.GetHeader("X-Watch-Instant-Sw-Close"))
		overrideString(&p.WatchType, c.GetHeader("X-Watch-Type"))

		if rawSDK := strings.TrimSpace(c.GetHeader("X-Watch-SDK-Int")); rawSDK != "" {
			v, err := strconv.Atoi(rawSDK)
			if err != nil {
				response.AbortWithError(c, domain.NewInvalidInputError(
					fmt.Sprintf("invalid header X-Watch-SDK-Int=%q: must be an integer", rawSDK),
					map[string]any{"header": "X-Watch-SDK-Int"},
				))
				return
			}
			p.SDKInt = v
		}

		if rawPlat := strings.TrimSpace(c.GetHeader("X-Watch-Plat-Code")); rawPlat != "" {
			v, err := strconv.Atoi(rawPlat)
			if err != nil {
				response.AbortWithError(c, domain.NewInvalidInputError(
					fmt.Sprintf("invalid header X-Watch-Plat-Code=%q: must be an integer", rawPlat),
					map[string]any{"header": "X-Watch-Plat-Code"},
				))
				return
			}
			p.PlatCode = v
		}

		if tok := strings.TrimSpace(c.GetHeader("X-HeyTap-Token")); tok != "" {
			p.Token = tok
		} else if auth := strings.TrimSpace(c.GetHeader("Authorization")); len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
			p.Token = strings.TrimSpace(auth[7:])
		}

		if err := p.Validate(); err != nil {
			response.AbortWithError(c, err)
			return
		}

		c.Set(contextKeyDeviceProfile, p)
		c.Next()
	}
}

// GetDeviceProfile retrieves the validated DeviceProfile from gin.Context.
func GetDeviceProfile(c *gin.Context) domain.DeviceProfile {
	if v, ok := c.Get(contextKeyDeviceProfile); ok {
		if p, ok := v.(domain.DeviceProfile); ok {
			return p
		}
	}
	return domain.DefaultDeviceProfile()
}

func overrideString(target *string, val string) {
	if trimmed := strings.TrimSpace(val); trimmed != "" {
		*target = trimmed
	}
}
