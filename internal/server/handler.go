package server

import (
	"net/http"
	"time"

	"github.com/kobaltio/api/internal/utils"
)

func (s *Server) ConvertHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		s.logger.Info("conversion completed", "duration", time.Since(start).String())
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	url := r.URL.Query().Get("url")
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")

	if url == "" || title == "" || artist == "" {
		s.SendErr(w, "missing required query params")
		return
	}

	s.SendRes(w, StatusProgress, "Validating YouTube URL...", 10)
	if !utils.IsValidURL(url) {
		s.SendErr(w, "invalid YouTube link")
	}

	s.SendRes(w, StatusProgress, "Validating video duration...", 20)
	duration, err := utils.GetVideoDuration(url)
	if err != nil {
		s.SendErr(w, "error getting video duration")
		return
	}
	if duration > (5 * time.Minute) {
		s.SendErr(w, "video is longer than 5 minutes")
		return
	}

	s.SendRes(w, StatusProgress, "Getting thumbnail URL...", 40)
	thumbnailURL, err := utils.GetThumbnailURL(url)
	if err != nil {
		s.SendErr(w, "error getting thumbnail URL")
		return
	}

	s.SendRes(w, StatusProgress, "Downloading thumbnail...", 50)
	thumbnail, err := utils.DownloadThumbnail(thumbnailURL)
	if err != nil {
		s.SendErr(w, "error downloading thumbnail")
		return
	}

	s.SendRes(w, StatusProgress, "Cropping thumbnail...", 70)
	croppedCover, err := utils.CropCover(thumbnail)
	if err != nil {
		s.SendErr(w, "error cropping thumbnail")
		return
	}

	s.SendRes(w, StatusProgress, "Downloading and embedding audio...", 80)
	if err := utils.DownloadAudio(url, croppedCover, title, artist); err != nil {
		s.SendErr(w, "error downloading audio")
		return
	}

	s.SendRes(w, StatusCompleted, "Conversion completed", 100)
}
