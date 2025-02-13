package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kobaltio/api/internal/server"
	"go.uber.org/zap"
)

func gracefulShutdown(server *http.Server, done chan bool, logger *zap.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	logger.Info("shutting down gracefully, press Ctrl+C again to force")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exiting")

	done <- true
}

func main() {
	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") == "development" {
		logger = zap.Must(zap.NewDevelopment())
	}

	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("failed to sync logger: %s", err)
		}
	}()

	server := server.NewServer(logger)

	done := make(chan bool, 1)
	go gracefulShutdown(server, done, logger)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	logger.Info("graceful shutdown complete")
}
