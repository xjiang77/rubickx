package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	lab "rubickx/system-design/systems/01-rate-limiter/lab/server"
)

func main() {
	address := envOrDefault("LAB_ADDR", "127.0.0.1:8080")
	root := os.Getenv("LAB_ROOT")
	redisAddress := os.Getenv("REDIS_ADDR")
	failurePolicy := envOrDefault("RATE_LIMIT_FAILURE_POLICY", "fail-open")
	app := lab.NewApp(lab.AppConfig{LabRoot: root, RedisAddr: redisAddress, FailurePolicy: failurePolicy})
	server := &http.Server{
		Addr:              address,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	log.Printf("Rate Limiter Lab listening on http://%s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
	if err := app.Close(); err != nil {
		log.Printf("debug session cleanup: %v", err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
