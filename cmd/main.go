package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/kobaltio/api/internal/convert"
)

func main() {
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
		middleware.Heartbeat("/healthz"),
		httprate.LimitByIP(10, time.Minute),
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/convert", convert.RegisterRoutes())
	})

	log.Fatal(http.ListenAndServe(":8080", router))
}
