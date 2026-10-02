package model

import "time"

type Telemetry struct {
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	CPUPct    float64   `json:"cpu_pct"`
	MemPct    float64   `json:"mem_pct"`
	DiskPct   float64   `json:"disk_pct"`
	NetIn     uint64    `json:"net_in"`
	NetOut    uint64    `json:"net_out"`
	EventType string    `json:"event_type"`
}
