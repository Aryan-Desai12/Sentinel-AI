package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"net/http"

	"sentinel-ai/agents/synthetic-monitor/internal/checker"
	"sentinel-ai/agents/synthetic-monitor/internal/config"
	"sentinel-ai/agents/synthetic-monitor/internal/delivery"
	"sentinel-ai/agents/synthetic-monitor/internal/model"
	"sentinel-ai/agents/synthetic-monitor/internal/scheduler"
	"sentinel-ai/agents/synthetic-monitor/internal/transport"
	"sentinel-ai/agents/synthetic-monitor/internal/worker"
)

func main() {
	// Logging

	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			nil,
		),
	)

	slog.SetDefault(logger)

	
	// Context / graceful shutdown
	
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

		// Load configuration

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error(
			"failed to load configuration",
			"service", "synthetic-monitor",
			"error", err,
		)

		os.Exit(1)
	}

	// Gateway API key
	
	apiKey := os.Getenv("SENTINEL_GATEWAY_API_KEY")

	if apiKey == "" {
		slog.Error(
			"gateway API key is missing",
			"service", "synthetic-monitor",
			"env", "SENTINEL_GATEWAY_API_KEY",
		)

		os.Exit(1)
	}

	// Shared SSRF-aware HTTP client

	resolver := net.DefaultResolver

// SSRF-aware client for checking configured targets.
checkerHTTPClient := transport.NewClient(resolver)

// Normal HTTP client for communicating with our Gateway.
gatewayHTTPClient := &http.Client{}

deliveryClient := delivery.NewClient(
	gatewayHTTPClient,
	cfg.Delivery.GatewayURL,
	apiKey,
)
	// Delivery buffer

	buffer := delivery.NewBuffer(
		cfg.Delivery.BufferCapacity,
		time.Duration(cfg.Delivery.BufferTTLSeconds)*time.Second,
	)
	
	// Delivery workers

	deliveryWorker := delivery.NewWorker(
		buffer,
		deliveryClient,
	)

	var deliveryWG sync.WaitGroup

	for i := 0; i < cfg.Delivery.Workers; i++ {
		deliveryWG.Add(1)

		go func() {
			defer deliveryWG.Done()
			deliveryWorker.Run(ctx)
		}()
	}

	// Checker

	checkerClient := checker.New(checkerHTTPClient)

	// Scheduler jobs

	jobs := make(chan scheduler.Job, 100)

	// Check result → event → delivery buffer

	onResult := func(result checker.Result) {
		event := model.FromResult(result)
		buffer.Enqueue(event)
	}

	// Check worker pool


	checkPool := worker.New(
		cfg.Check.Workers,
		cfg.Check.MaxInFlight,
		jobs,
		checkerClient.Check,
		onResult,
	)

	var checkWG sync.WaitGroup

	checkWG.Add(1)

	go func() {
		defer checkWG.Done()
		checkPool.Run(ctx)
	}()

	// Start schedulers

	var schedulerWG sync.WaitGroup

	for _, target := range cfg.Targets {
		s := scheduler.New(target, jobs)

		schedulerWG.Add(1)

		go func() {
			defer schedulerWG.Done()
			s.Run(ctx)
		}()
	}

	slog.Info(
		"synthetic monitor started",
		"service", "synthetic-monitor",
		"targets", len(cfg.Targets),
		"check_workers", cfg.Check.Workers,
		"max_in_flight", cfg.Check.MaxInFlight,
		"delivery_workers", cfg.Delivery.Workers,
	)

	// Wait for shutdown signal

	<-ctx.Done()

	slog.Info(
		"shutdown signal received",
		"service", "synthetic-monitor",
	)

	// Shutdown
	
	// Stop schedulers and check workers first.
	schedulerWG.Wait()
	checkWG.Wait()

	// Delivery workers observe the same cancelled context
	// and stop after cancellation.
	deliveryWG.Wait()

	slog.Info(
		"synthetic monitor stopped",
		"service", "synthetic-monitor",
	)
}

//                     config.yaml
//                          │
//                          ↓
//                     main.go
//                          │
//        ┌─────────────────┼─────────────────┐
//        │                 │                 │
//        ↓                 ↓                 ↓
//    Scheduler          Checker          Delivery
//        │                 │                 │
//        ↓                 │                 ↓
//    jobs channel           │             Buffer
//        │                 │                 │
//        ↓                 ↓                 ↓
//    Worker Pool ───────→ Result ─────→ Delivery Worker
//        │                                   │
//        ↓                                   ↓
//  Per-target CAS                       Delivery Client
//        │                                   │
//        ↓                                   ↓
//  Global semaphore                    SSRF Transport
//        │                                   │
//        ↓                                   ↓
//     HTTP check                         Gateway