package agent

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/buffer"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/collector"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/transport"
)

//Think of agent.go as the orchestrator of the telemetry agent.
// It doesn't know how CPU, memory, disk, or network are collected. It only decides when to collect them and manages the agent's lifecycle.
//agent.go: What does the application do while it is running?

//main.go manages the overall application's lifecycle, while agent.go manages the telemetry agent's collection lifecycle and triggers collectors at the configured interval.

//One physical host = one node ID; many telemetry events = repeated measurements from that node.
type Agent struct {
	nodeID   string
	interval time.Duration

	cpu     *collector.CPUCollector
	network *collector.NetworkCollector
	buffer  *buffer.Buffer

	transport *transport.Transport
}

func New(
	nodeID string,
	interval time.Duration,
	gatewayURL string,
	apiKey string,
) *Agent {
	return &Agent{
		nodeID:   nodeID,
		interval: interval,

		cpu:     collector.NewCPUCollector(),
		network: collector.NewNetworkCollector(),
		buffer:  buffer.New(100),

		transport: transport.New(gatewayURL, apiKey),
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
		Timestamp: time.Now().UTC(),
		CPUPct:    cpu,
		MemPct:    memory,
		DiskPct:   disk,
		NetIn:     network.Received,
		NetOut:    network.Transmitted,
	}

	if err := event.Validate(); err != nil {
		log.Printf("telemetry validation failed: %v", err)
		return
	}

	if err := a.buffer.Add(event); err != nil {
		log.Printf(
			"telemetry buffer full; dropping event: node_id=%s timestamp=%s",
			event.NodeID,
			event.Timestamp.Format(time.RFC3339),
		)
		return
	}

	log.Printf("telemetry buffered: node_id=%s", event.NodeID)
}

func (a *Agent) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("telemetry transport stopped")
			return
		default:
		}

		event, ok := a.buffer.Peek()
		if !ok {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if a.isStale(event) {
			if _, ok := a.buffer.Remove(); !ok {
				log.Printf("failed to remove stale telemetry")
				continue
			}

			log.Printf(
				"dropped stale telemetry: node_id=%s timestamp=%s",
				event.NodeID,
				event.Timestamp.Format(time.RFC3339),
			)

			continue
		}

		if err := a.sendWithRetry(ctx, event); err != nil {
			var transportErr *transport.Error

			if errors.As(err, &transportErr) &&
				(transportErr.StatusCode == http.StatusBadRequest ||
					transportErr.StatusCode == http.StatusUnauthorized) {

				if _, ok := a.buffer.Remove(); !ok {
					log.Printf("failed to remove permanently rejected telemetry")
					continue
				}

				log.Printf(
					"dropped permanently rejected telemetry: node_id=%s timestamp=%s error=%v",
					event.NodeID,
					event.Timestamp.Format(time.RFC3339),
					err,
				)

				continue
			}

			log.Printf(
				"telemetry delivery stopped for event node_id=%s: %v",
				event.NodeID,
				err,
			)
			continue
		}

		if _, ok := a.buffer.Remove(); !ok {
			log.Printf("failed to remove successfully sent telemetry")
			continue
		}

		log.Printf(
			"telemetry sent: node_id=%s timestamp=%s",
			event.NodeID,
			event.Timestamp.Format(time.RFC3339),
		)
	}
}

func isRetryable(err error) bool {
	var transportErr *transport.Error

	if errors.As(err, &transportErr) {
		switch transportErr.StatusCode {
		case http.StatusBadRequest,
			http.StatusUnauthorized:
			return false
		}
	}

	return true
}

func (a *Agent) sendWithRetry(
	ctx context.Context,
	event model.Telemetry,
) error {
	backoff := time.Second

	for {
		err := a.transport.Send(ctx, event)

		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		log.Printf(
			"telemetry send failed, retrying in %s: %v",
			backoff,
			err,
		)

		timer := time.NewTimer(backoff)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

const telemetryTTL = 5 * time.Minute

func (a *Agent) isStale(event model.Telemetry) bool {
	return time.Since(event.Timestamp) > telemetryTTL
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
