package read

import (
	"context"
	"time"
)

type Incident struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type IncidentsRepository interface {
	GetIncidents(
		ctx context.Context,
		from time.Time,
		to time.Time,
	) ([]Incident, error)
}
