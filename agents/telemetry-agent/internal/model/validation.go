package model

//Collectors read directly from the operating system. We should not assume every value is valid.Only valid telemetry is allowed to enter the outbound pipeline.
//We don't want an obviously invalid event entering the pipeline.
//We're allowing a small future tolerance because system clocks can have minor differences.

import (
	"fmt"
	"time"
)

func (t Telemetry) Validate() error {
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

	if t.Timestamp.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("timestamp cannot be too far in the future")
	}

	return nil
}
