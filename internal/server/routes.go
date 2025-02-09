package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/kobaltio/api/internal/converter"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))
	r.Use(httprate.LimitByIP(10, time.Minute))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/v1/convert", s.convertHandler)

	return r
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

func (s *Server) convertHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	requestID := middleware.GetReqID(r.Context())
	workDir := filepath.Join(os.TempDir(), "kobalt", requestID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	if err := os.MkdirAll(workDir, 0755); err != nil {
		sendErr(w, "failed to create temp directory")
		return
	}

	go func() {
		<-ctx.Done()
		os.RemoveAll(workDir)
	}()

	ctx = context.WithValue(ctx, converter.WorkDir, workDir)

	url := r.URL.Query().Get("url")
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")

	if url == "" || title == "" || artist == "" {
		sendErr(w, "missing required query params")
		return
	}

	sendRes(w, StatusProgress, "Validating YouTube URL...", 10)
	if !converter.IsValidURL(url) {
		sendErr(w, "invalid YouTube link")
	}

	sendRes(w, StatusProgress, "Validating video duration...", 20)
	duration, err := converter.GetVideoDuration(url)
	if err != nil {
		sendErr(w, "error getting video duration")
		return
	}
	if duration > (5 * time.Minute) {
		sendErr(w, "video is longer than 5 minutes")
		return
	}

	sendRes(w, StatusProgress, "Downloading audio and thumbnail...", 70)

	audioErrChan := make(chan error)
	coverErrChan := make(chan error)

	go func() {
		audioErrChan <- converter.DownloadAudio(ctx, url)
	}()
	go func() {
		coverErrChan <- converter.DownloadCover(ctx, url)
	}()

	audioErr := <-audioErrChan
	coverErr := <-coverErrChan

	if audioErr != nil {
		cancel()
		sendErr(w, "error downloading audio")
		return
	}
	if coverErr != nil {
		cancel()
		sendErr(w, "error downloading thumbnail")
		return
	}

	sendRes(w, StatusProgress, "Embedding mp3 file...", 90)
	if err := converter.EmbedAudio(ctx, title, artist); err != nil {
		cancel()
		sendErr(w, "error embedding mp3 file")
		return
	}

	sendRes(w, StatusCompleted, "Conversion completed", 100)
	os.RemoveAll(workDir)
}

func sendRes(w http.ResponseWriter, status Status, msg string, progress int) {
	data, _ := json.Marshal(Response{
		Status:   status,
		Message:  msg,
		Progress: progress,
	})

	fmt.Fprintf(w, "data: %s\n\n", data)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func sendErr(w http.ResponseWriter, msg string) {
	sendRes(w, StatusError, msg, 0)
}
