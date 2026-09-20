package model

//This is Synthetic Check Event Model

import (
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/checker"
)

type SyntheticCheckEvent struct {
	URL        string    `json:"url"`
	Service    string    `json:"service,omitempty"`
	Status     string    `json:"status"`
	LatencyMs  int64     `json:"latency_ms"`
	TTFBMs     int64     `json:"ttfb_ms"`
	HTTPStatus int       `json:"http_status"`
	ErrorType  string    `json:"error_type,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func FromResult(result checker.Result) SyntheticCheckEvent {
	return SyntheticCheckEvent{
		URL:        result.URL,
		Service:    result.Service,
		Status:     result.Status,
		LatencyMs:  result.LatencyMs,
		TTFBMs:     result.TTFBMs,
		HTTPStatus: result.HTTPStatus,
		ErrorType:  result.ErrorType,
		Timestamp:  result.Timestamp,
	}
}
