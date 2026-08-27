package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/agent"
	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	telemetryAgent := agent.New(
		cfg.NodeID,
		cfg.IntervalSeconds,
	)

	log.Printf(
		"Sentinel Telemetry Agent started: node_id=%s interval=%ds",
		cfg.NodeID,
		cfg.IntervalSeconds,
	)

	go telemetryAgent.Run(ctx)

	<-ctx.Done()

	log.Println("Shutdown signal received")
	log.Println("Sentinel Telemetry Agent stopped")
}

// basic pipeline working:
//
// config.yaml
//     ↓
// config.Load()
//     ↓
// main.go
//     ↓
// agent.New()
//     ↓
// agent.Run()
//     ↓
// every 10 seconds
//     ↓
// CPU ─ Memory ─ Disk ─ Network
//

//	gofmt → runs Go's formatter.
// -w → write the formatted result back into the file.
// agent.go → file to format.
// It does not change your program's logic. It primarily standardizes formatting such as indentation, spacing, and some syntactic layout.