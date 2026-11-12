package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"geofence-backend/internal/api"
	"geofence-backend/internal/db"
	"geofence-backend/internal/ws"
)

// getEnv returns the environment variable, or the fallback when it is not set.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "postgres://geo:geo@localhost:5432/geofence?sslmode=disable")
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "*")

	rateLimit, err := strconv.Atoi(getEnv("LOCATION_RATE_LIMIT_PER_MIN", "120"))
	if err != nil {
		log.Fatalf("LOCATION_RATE_LIMIT_PER_MIN must be a number: %v", err)
	}

	// ctx is cancelled when we get Ctrl+C or a stop signal from Docker.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	hub := ws.NewHub()
	server := api.NewServer(pool, hub, api.Options{
		AllowedOrigins:          allowedOrigins,
		LocationRateLimitPerMin: rateLimit,
	})

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown: finish running requests before exiting.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("API listening on :%s", port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}
