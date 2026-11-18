package service

import (
	"context"

	"owas-api/internal/domain"
)

// HealthUseCase defines the application-layer contract for Health Cloud operations.
type HealthUseCase interface {
	QuerySwitchStatus(ctx context.Context, profile domain.DeviceProfile, switchType int) (*domain.SwitchStatusResponse, error)
	CheckPay(ctx context.Context, profile domain.DeviceProfile, input domain.CheckPayInput) (*domain.CheckPayResponse, error)
	ExchangeSymmetricKey(ctx context.Context, profile domain.DeviceProfile, input domain.ExchangeKeyInput) (*domain.ExchangeKeyResponse, error)
}

var _ HealthUseCase = (*HealthService)(nil)

// HealthService orchestrates validation and gateway calls for Health Cloud APIs.
type HealthService struct {
	gateway domain.HealthGateway
}

// NewHealthService constructs a HealthService with the injected HealthGateway.
func NewHealthService(gateway domain.HealthGateway) *HealthService {
	return &HealthService{gateway: gateway}
}

func (s *HealthService) QuerySwitchStatus(ctx context.Context, profile domain.DeviceProfile, switchType int) (*domain.SwitchStatusResponse, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if switchType <= 0 {
		return nil, domain.NewInvalidInputError("path parameter 'switchType' must be > 0", map[string]any{"field": "switchType"})
	}
	return s.gateway.QuerySwitchStatus(ctx, dev, switchType)
}

func (s *HealthService) CheckPay(ctx context.Context, profile domain.DeviceProfile, input domain.CheckPayInput) (*domain.CheckPayResponse, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if input.AppID <= 0 {
		return nil, domain.NewInvalidInputError("'appId' must be > 0", map[string]any{"field": "appId"})
	}
	return s.gateway.CheckPay(ctx, dev, input)
}

func (s *HealthService) ExchangeSymmetricKey(ctx context.Context, profile domain.DeviceProfile, input domain.ExchangeKeyInput) (*domain.ExchangeKeyResponse, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	alg := 4
	if input.EncryptionAlgorithm != nil {
		alg = *input.EncryptionAlgorithm
	}
	first := 0
	if input.FirstExchangeKey != nil {
		first = *input.FirstExchangeKey
	}
	if alg <= 0 {
		return nil, domain.NewInvalidInputError("'encryptionAlgorithm' must be > 0", map[string]any{"field": "encryptionAlgorithm"})
	}
	if first < 0 {
		return nil, domain.NewInvalidInputError("'firstExchangeKey' must be >= 0", map[string]any{"field": "firstExchangeKey"})
	}
	return s.gateway.ExchangeSymmetricKey(ctx, dev, alg, first)
}
