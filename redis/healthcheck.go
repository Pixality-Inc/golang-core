package redis

import (
	"context"
	"time"
)

type HealthcheckService struct {
	client Client
}

func NewHealthcheckService(client Client) *HealthcheckService {
	return &HealthcheckService{
		client: client,
	}
}

func (h *HealthcheckService) IsOK() bool {
	if !h.client.IsConnected() {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testKey := "__healthcheck__"
	testValue := "ok"

	if err := h.client.SetKey(ctx, testKey, testValue, 10*time.Second); err != nil {
		return false
	}

	value, err := h.client.GetString(ctx, testKey)
	if err != nil {
		return false
	}

	return value == testValue
}
