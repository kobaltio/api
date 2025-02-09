package converter

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/webp"
)

func IsValidURL(url string) bool {
	regex := `^(https?://)?(www\.)?(youtube\.com|youtu\.be)/.+$`
	matched, err := regexp.MatchString(regex, url)
	return matched && err == nil
}

func GetVideoDuration(url string) (time.Duration, error) {
	cmd := exec.Command("yt-dlp", "--get-duration", url)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	durationStr := strings.TrimSpace(string(output))
	return parseDuration(durationStr)
}

func parseDuration(duration string) (time.Duration, error) {
	parts := strings.Split(duration, ":")
	if len(parts) > 2 {
		return 0, errors.New("unexpected duration format")
	}

	seconds := 0
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return 0, errors.New("invalid number in duration")
		}
		seconds = seconds*60 + value
	}

	return time.Duration(seconds) * time.Second, nil
}

func GetThumbnailURL(url string) (string, error) {
	cmd := exec.Command("yt-dlp", "--get-thumbnail", url)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func DownloadThumbnail(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func CropCover(imgData []byte) ([]byte, error) {
	img, err := webp.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	size := width
	if height < width {
		size = height
	}
	x := (width - size) / 2
	y := (height - size) / 2
	cropRect := image.Rect(0, 0, size, size)

	croppedImg := image.NewRGBA(cropRect)
	draw.Draw(croppedImg, cropRect, img, image.Pt(x, y), draw.Src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, croppedImg, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DownloadAudio(url string, imgData []byte, title string, artist string) error {
	if err := os.WriteFile("cover.jpg", imgData, 0644); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}
	defer os.Remove("cover.jpg")

	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", "%(title)s.%(ext)s",
		"--no-keep-video",
		url,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}

	files, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var mp3File string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".mp3") {
			mp3File = file.Name()
			break
		}
	}

	if mp3File == "" {
		return errors.New("no MP3 file found after download")
	}

	cmd = exec.Command(
		"ffmpeg",
		"-i", mp3File,
		"-i", "cover.jpg",
		"-map", "0:0",
		"-map", "1:0",
		"-metadata", "title="+title,
		"-metadata", "artist="+artist,
		"-c:a", "copy",
		"-c:v", "copy",
		"-id3v2_version", "3",
		"-disposition:v:0", "attached_pic",
		"output.mp3",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add metadata: %w", err)
	}

	if err := os.Remove(mp3File); err != nil {
		return fmt.Errorf("failed to remove original mp3 file: %w", err)
	}

	return nil
}
