package read

import (
	"context"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
)

type MetricsRepository interface {
	GetHistory(
		ctx context.Context,
		nodeID string,
		from time.Time,
		to time.Time,
	) ([]model.Telemetry, error)
}
