package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	port   int
	logger *slog.Logger
}

type Status string

const (
	StatusProgress  Status = "progress"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type Response struct {
	Status   Status `json:"status"`
	Message  string `json:"message"`
	Progress int    `json:"progress,omitempty"`
	Error    string `json:"error,omitempty"`
}

func NewServer(logger *slog.Logger) *http.Server {
	newServer := &Server{
		port:   8080,
		logger: logger,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", newServer.port),
		Handler:      newServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/convert", s.ConvertHandler)
	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) SendRes(w http.ResponseWriter, status Status, msg string, progress int) {
	data, _ := json.Marshal(Response{
		Status:   status,
		Message:  msg,
		Progress: progress,
	})

	_, err := fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		s.logger.Error("client disconnected")
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) SendErr(w http.ResponseWriter, msg string) {
	s.SendRes(w, StatusError, msg, 0)
}
