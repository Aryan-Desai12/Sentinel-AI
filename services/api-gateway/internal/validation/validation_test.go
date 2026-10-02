package validation

import (
	"testing"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
)

func validTelemetry() model.Telemetry {
	return model.Telemetry{
		NodeID:    "node-01",
		Timestamp: time.Now(),
		CPUPct:    45.5,
		MemPct:    60.2,
		DiskPct:   30.1,
		NetIn:     1000,
		NetOut:    2000,
	}
}

func TestValidateTelemetry(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*model.Telemetry)
		wantErr bool
	}{
		{
			name:    "valid telemetry",
			modify:  func(t *model.Telemetry) {},
			wantErr: false,
		},
		{
			name: "missing node_id",
			modify: func(t *model.Telemetry) {
				t.NodeID = ""
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			modify: func(t *model.Telemetry) {
				t.Timestamp = time.Time{}
			},
			wantErr: true,
		},
		{
			name: "cpu below zero",
			modify: func(t *model.Telemetry) {
				t.CPUPct = -1
			},
			wantErr: true,
		},
		{
			name: "cpu above 100",
			modify: func(t *model.Telemetry) {
				t.CPUPct = 101
			},
			wantErr: true,
		},
		{
			name: "memory below zero",
			modify: func(t *model.Telemetry) {
				t.MemPct = -1
			},
			wantErr: true,
		},
		{
			name: "memory above 100",
			modify: func(t *model.Telemetry) {
				t.MemPct = 101
			},
			wantErr: true,
		},
		{
			name: "disk below zero",
			modify: func(t *model.Telemetry) {
				t.DiskPct = -1
			},
			wantErr: true,
		},
		{
			name: "disk above 100",
			modify: func(t *model.Telemetry) {
				t.DiskPct = 101
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry := validTelemetry()
			tt.modify(&telemetry)

			err := ValidateTelemetry(telemetry)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTelemetry() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}