package main

import (
	"log"
	"net/http"

	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/balancer"
	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/logger"
	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/proxy"
)

func main() {
	appLogger := logger.New("load-balancer")

	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	rr, err := balancer.NewRoundRobin(backends)
	if err != nil {
		appLogger.Error("load_balancer_init_failed", map[string]any{
			"error": err.Error(),
		})
		log.Fatal(err)
	}

	rr.Logger = appLogger
	rr.StartHealthChecks()

	reverseProxy := &proxy.Proxy{
		Balancer: rr,
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: reverseProxy,
	}

	appLogger.Info("load_balancer_started", map[string]any{
		"port":     8080,
		"backends": len(backends),
	})

	if err := server.ListenAndServe(); err != nil {
		appLogger.Error("load_balancer_stopped", map[string]any{
			"error": err.Error(),
		})
		log.Fatal(err)
	}
}
