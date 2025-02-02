package convert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kobaltio/api/internal/utils"
)

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

func RegisterRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", ConvertHandler)
	return r
}

func ConvertHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	url := r.URL.Query().Get("url")
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")

	if url == "" || title == "" || artist == "" {
		sendErr(w, "missing required query params")
		return
	}

	sendRes(w, StatusProgress, "Validating YouTube URL...", 10)
	if !utils.IsValidURL(url) {
		sendErr(w, "invalid YouTube link")
	}

	sendRes(w, StatusProgress, "Validating video duration...", 20)
	duration, err := utils.GetVideoDuration(url)
	if err != nil {
		sendErr(w, "error getting video duration")
		return
	}
	if duration > (5 * time.Minute) {
		sendErr(w, "video is longer than 5 minutes")
		return
	}

	sendRes(w, StatusProgress, "Getting thumbnail URL...", 40)
	thumbnailURL, err := utils.GetThumbnailURL(url)
	if err != nil {
		sendErr(w, "error getting thumbnail URL")
		return
	}

	sendRes(w, StatusProgress, "Downloading thumbnail...", 50)
	thumbnail, err := utils.DownloadThumbnail(thumbnailURL)
	if err != nil {
		sendErr(w, "error downloading thumbnail")
		return
	}

	sendRes(w, StatusProgress, "Cropping thumbnail...", 70)
	croppedCover, err := utils.CropCover(thumbnail)
	if err != nil {
		sendErr(w, "error cropping thumbnail")
		return
	}

	sendRes(w, StatusProgress, "Downloading and embedding audio...", 80)
	if err := utils.DownloadAudio(url, croppedCover, title, artist); err != nil {
		sendErr(w, "error downloading audio")
		return
	}

	sendRes(w, StatusCompleted, "Conversion completed", 100)
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
