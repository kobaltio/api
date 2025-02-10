package converter

import (
	"testing"
	"time"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid youtube.com watch URL",
			url:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "valid mobile youtube URL",
			url:  "https://m.youtube.com/watch?v=dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "valid embed URL",
			url:  "https://youtube.com/embed/dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "valid v URL",
			url:  "https://youtube.com/v/dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "valid youtu.be short URL",
			url:  "https://youtu.be/dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "valid URL without protocol",
			url:  "www.youtube.com/watch?v=dQw4w9WgXcQ",
			want: true,
		},
		{
			name: "empty URL",
			url:  "",
			want: false,
		},
		{
			name: "invalid domain",
			url:  "https://notyoutube.com/watch?v=dQw4w9WgXcQ",
			want: false,
		},
		{
			name: "invalid video ID",
			url:  "https://youtube.com/watch?v=123",
			want: false,
		},
		{
			name: "playlist URL",
			url:  "https://youtube.com/playlist?list=PLdEKZTDH6MLEDYWRQhQ53Bl7e0o71_l0I",
			want: false,
		},
		{
			name: "channel URL",
			url:  "https://youtube.com/channel/UCuAXFkgsw1L7xaCfnd5JJOw",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidURL(tt.url); got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     time.Duration
		wantErr  bool
	}{
		{
			name:     "valid minutes and seconds",
			duration: "3:45",
			want:     3*time.Minute + 45*time.Second,
			wantErr:  false,
		},
		{
			name:     "valid zero minutes",
			duration: "0:30",
			want:     30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "invalid hours:minutes:seconds format",
			duration: "1:30:45",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "invalid empty string",
			duration: "",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "invalid format with letters",
			duration: "3:ab",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "invalid empty minutes",
			duration: ":45",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{
			name: "jpeg extension",
			file: "image.jpeg",
			want: "jpeg",
		},
		{
			name: "uppercase JPEG",
			file: "image.JPEG",
			want: "jpeg",
		},
		{
			name: "webp extension",
			file: "image.webp",
			want: "webp",
		},
		{
			name: "url with query params",
			file: "https://example.com/image.jpeg?size=large",
			want: "jpeg",
		},
		{
			name: "no extension",
			file: "filename",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFileExtension(tt.file); got != tt.want {
				t.Errorf("getFileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}
