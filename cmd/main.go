package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/kobaltio/api/internal/convert"
)

func gracefulShutdown(server *http.Server) {
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 10*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-serverCtx.Done()
}

func registerRoutes() *chi.Mux {
	router := chi.NewRouter()

	router.Use(
		cors.Handler(
			cors.Options{
				AllowedMethods: []string{"GET", "OPTIONS"},
			},
		),
		middleware.RequestID,
		middleware.Logger,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.Timeout(30*time.Second),
		middleware.Heartbeat("/healthz"),
		httprate.LimitByIP(10, time.Minute),
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/convert", convert.RegisterRoutes())
	})

	return router
}

func main() {
	gracefulShutdown(&http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: registerRoutes(),
	})
}
