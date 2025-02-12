package converter

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/image/webp"
)

type Key string

const WorkDir Key = "workDir"

const S3BucketName = "kobaltio-files"

func IsValidURL(url string) bool {
	regex := `^(?:https?:\/\/)?(?:m\.|www\.)?(?:youtu\.be\/|youtube\.com\/(?:embed\/|v\/|watch\?v=|watch\?.+&v=))((\w|-){11})(?:\S+)?$`
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

func DownloadCover(ctx context.Context, url string) error {
	workDir := ctx.Value(WorkDir).(string)

	thumbnailURL, err := getThumbnailURL(url)
	if err != nil {
		return err
	}

	imgData, err := getThumbnailData(thumbnailURL)
	if err != nil {
		return err
	}

	ext := getFileExtension(thumbnailURL)
	croppedImgData, err := cropThumbnail(imgData, ext)
	if err != nil {
		return err
	}

	output := filepath.Join(workDir, "cover.jpg")
	return os.WriteFile(output, croppedImgData, 0644)
}

func DownloadAudio(ctx context.Context, url string) error {
	workDir := ctx.Value(WorkDir).(string)
	output := filepath.Join(workDir, "audio.mp3")

	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", output,
		"--no-keep-video",
		url,
	)
	return cmd.Run()
}

func EmbedAudio(ctx context.Context, title, artist, fileName string) error {
	workDir := ctx.Value(WorkDir).(string)

	mp3File := filepath.Join(workDir, "audio.mp3")
	coverFile := filepath.Join(workDir, "cover.jpg")
	outputFile := filepath.Join(workDir, fileName)

	cmd := exec.Command(
		"ffmpeg",
		"-i", mp3File,
		"-i", coverFile,
		"-map", "0:0",
		"-map", "1:0",
		"-metadata", "title="+title,
		"-metadata", "artist="+artist,
		"-c:a", "copy",
		"-c:v", "copy",
		"-id3v2_version", "3",
		"-disposition:v:0", "attached_pic",
		outputFile,
	)

	return cmd.Run()
}

func UploadToS3(ctx context.Context, fileName string) (string, error) {
	workDir := ctx.Value(WorkDir).(string)
	path := filepath.Join(workDir, fileName)

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-north-1"
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return "", err
	}

	svc := s3.New(sess)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	key := filepath.Base(path)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(S3BucketName),
		Key:    aws.String(key),
		Body:   file,
		ACL:    aws.String("private"),
	})
	if err != nil {
		return "", err
	}

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(S3BucketName),
		Key:    aws.String(key),
	})

	urlStr, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", err
	}

	return urlStr, nil
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

func getThumbnailURL(url string) (string, error) {
	cmd := exec.Command("yt-dlp", "--get-thumbnail", url)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getThumbnailData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func cropThumbnail(imgData []byte, ext string) ([]byte, error) {
	var img image.Image
	bytesReader := bytes.NewReader(imgData)

	switch ext {
	case "webp":
		image, err := webp.Decode(bytesReader)
		if err != nil {
			return nil, err
		}
		img = image
	default:
		image, _, err := image.Decode(bytesReader)
		if err != nil {
			return nil, err
		}
		img = image
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

func getFileExtension(url string) string {
	baseURL := strings.Split(url, "?")[0]
	baseURL = strings.Split(baseURL, "#")[0]

	ext := ""
	lastDot := strings.LastIndex(baseURL, ".")
	if lastDot != -1 && lastDot < len(baseURL)-1 {
		ext = baseURL[lastDot+1:]
	}
	return strings.ToLower(ext)
}
