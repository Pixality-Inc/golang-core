package postgres

import (
	"context"
	"time"
)

type HealthcheckService struct {
	database Database
}

func NewHealthcheckService(database Database) *HealthcheckService {
	return &HealthcheckService{
		database: database,
	}
}

func (h *HealthcheckService) IsOK() bool {
	if !h.database.IsConnected() {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	executor, err := h.database.Executor()
	if err != nil {
		return false
	}

	rows, err := executor.Query(ctx, "SELECT 1")
	if err != nil {
		return false
	}
	defer rows.Close()

	return rows.Next()
}
