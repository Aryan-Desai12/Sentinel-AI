package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/auth"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/dedupe"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/handler"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/logger"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/node"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/publisher"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/ratelimit"
)

func main() {
	appLogger := logger.New("api-gateway")

	db, err := sql.Open(
		"pgx",
		"postgres://sentinel:sentinel@localhost:5432/sentinel",
	)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	defer db.Close()

	appLogger.Info("postgres_connected", nil)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	defer redisClient.Close()

	appLogger.Info("redis_connected", nil)

	dedupeStore := dedupe.NewStore(
		redisClient,
		24*time.Hour,
	)

	rateLimiter := ratelimit.NewLimiter(
		redisClient,
		20,
	)

	pub, err := publisher.NewKafkaPublisher(
		[]string{"localhost:9092"},
	)
	if err != nil {
		log.Fatalf("failed to create Kafka publisher: %v", err)
	}

	defer pub.Close()

	repo := &node.Repository{
		DB: db,
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	healthHandler := handler.HealthHandler{
		DB:       db,
		Redis:    redisClient,
		Instance: port,
	}

	registerHandler := handler.RegisterHandler{
		Repository: repo,
		Logger:     appLogger,
	}

	metricsHandler := handler.MetricsHandler{
		Publisher:   pub,
		DedupeStore: dedupeStore,
		RateLimiter: rateLimiter,
		Logger:      appLogger,
	}

	syntheticHandler := handler.SyntheticHandler{
		Publisher: pub,
		Logger:    appLogger,
	}

	authMiddleware := auth.Middleware{
		Keys:   repo,
		Logger: appLogger,
	}

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/health",
		healthHandler.HandleHealth,
	)

	mux.HandleFunc(
		"/nodes/register",
		registerHandler.HandleRegister,
	)

	mux.Handle(
		"/metrics/node",
		authMiddleware.Authenticate(
			http.HandlerFunc(metricsHandler.HandleNodeMetrics),
		),
	)

	mux.Handle(
		"/checks/synthetic",
		authMiddleware.Authenticate(
			http.HandlerFunc(syntheticHandler.HandleSyntheticCheck),
		),
	)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	appLogger.Info("server_started", map[string]any{
		"port": port,
	})

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
