package model

import "time"

type SyntheticCheck struct {
	URL        string    `json:"url"`
	Service    string    `json:"service"`
	Status     string    `json:"status"`
	LatencyMs  int64     `json:"latency_ms"`
	TTFBMs     int64     `json:"ttfb_ms"`
	HTTPStatus int       `json:"http_status"`
	ErrorType  string    `json:"error_type"`
	Timestamp  time.Time `json:"timestamp"`
	EventType  string    `json:"event_type"`
}
