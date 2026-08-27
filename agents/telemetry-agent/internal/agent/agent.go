package agent

import (
	"context"
	"log"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/buffer"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/collector"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
)

//Think of agent.go as the orchestrator of the telemetry agent.
// It doesn't know how CPU, memory, disk, or network are collected. It only decides when to collect them and manages the agent's lifecycle.
//agent.go: What does the application do while it is running?

//main.go manages the overall application's lifecycle, while agent.go manages the telemetry agent's collection lifecycle and triggers collectors at the configured interval.

type Agent struct {
	nodeID   string
	interval time.Duration

	cpu     *collector.CPUCollector
	network *collector.NetworkCollector
	buffer  *buffer.Buffer
}

func New(nodeID string, intervalSeconds int) *Agent {
	return &Agent{
		interval: time.Duration(intervalSeconds) * time.Second,
		cpu:      collector.NewCPUCollector(),
		network:  collector.NewNetworkCollector(),
		buffer:   buffer.New(100),
	}
}

func (a *Agent) Run(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	log.Printf("telemetry collection started (interval=%s)", a.interval)

	for {
		select {
		case <-ticker.C:
			a.collect()

		case <-ctx.Done():
			log.Println("telemetry collection stopped")
			return
		}
	}
}

func (a *Agent) collect() {
	cpu, err := a.cpu.Collect()
	if err != nil {
		log.Printf("CPU collection failed: %v", err)
		return
	}

	memory, err := collector.CollectMemory()
	if err != nil {
		log.Printf("memory collection failed: %v", err)
		return
	}

	disk, err := collector.CollectDisk()
	if err != nil {
		log.Printf("disk collection failed: %v", err)
		return
	}

	network, err := a.network.Collect()
	if err != nil {
		log.Printf("network collection failed: %v", err)
		return
	}

	event := model.Telemetry{
		NodeID:    a.nodeID,
		Timestamp: time.Now(),
		CPUPct:    cpu,
		MemPct:    memory,
		DiskPct:   disk,
		NetIn:     network.Received,
		NetOut:    network.Transmitted,
	}

	if err := a.buffer.Add(event); err != nil {
		log.Printf("telemetry buffer error: %v", err)
		return
	}

	log.Printf("telemetry buffered: node_id=%s", event.NodeID)
}

// The flow of telemetry collection and validation can be visualized as follows:
// Collectors
//     ↓
// Telemetry struct
//     ↓
// Validate()
//     │
//     ├── invalid → discard + log
//     │
//     └── valid
//           ↓
//        next layer
//
//

