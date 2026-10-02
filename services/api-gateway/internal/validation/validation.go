package validation

import (
	"fmt"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
)

func ValidateTelemetry(t model.Telemetry) error {
	if t.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if t.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	if t.CPUPct < 0 || t.CPUPct > 100 {
		return fmt.Errorf("cpu_pct must be between 0 and 100")
	}

	if t.MemPct < 0 || t.MemPct > 100 {
		return fmt.Errorf("mem_pct must be between 0 and 100")
	}

	if t.DiskPct < 0 || t.DiskPct > 100 {
		return fmt.Errorf("disk_pct must be between 0 and 100")
	}

	return nil
}