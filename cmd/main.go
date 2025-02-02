package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kobaltio/api/internal/server"
)

func gracefulShutdown(server *http.Server, done chan bool, logger *slog.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	logger.Info("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown with error", "error", err)
	}

	logger.Info("server exiting")

	done <- true
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
