package youtube

import (
	"errors"
	"image"
	"testing"

	yt "github.com/kkdai/youtube/v2"
)

func TestValidateURL(t *testing.T) {
	ok := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=dQw4w9WgXcQ",
		"  https://m.youtube.com/watch?v=dQw4w9WgXcQ  ",
	}
	bad := []string{"", "youtube.com/watch?v=x", "https://vimeo.com/1", "ftp://youtube.com/x", "https://notyoutube.com/"}
	for _, u := range ok {
		if err := ValidateURL(u); err != nil {
			t.Errorf("%q: %v", u, err)
		}
	}
	for _, u := range bad {
		if !errors.Is(ValidateURL(u), ErrBadURL) {
			t.Errorf("%q должна быть отклонена", u)
		}
	}
}

func TestBestOpus(t *testing.T) {
	formats := yt.FormatList{
		{ItagNo: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, Bitrate: 130000},
		{ItagNo: 250, MimeType: `audio/webm; codecs="opus"`, Bitrate: 70000},
		{ItagNo: 251, MimeType: `audio/webm; codecs="opus"`, Bitrate: 135000},
		{ItagNo: 248, MimeType: `video/webm; codecs="vp9"`, Bitrate: 2000000},
	}
	f, err := bestOpus(formats)
	if err != nil || f.ItagNo != 251 {
		t.Fatalf("выбран %+v, %v; ожидали itag 251", f, err)
	}
	if _, err := bestOpus(formats[:1]); !errors.Is(err, errNoOpus) {
		t.Fatalf("без Opus ожидали errNoOpus, получили %v", err)
	}
}

func TestSquareCover(t *testing.T) {
	tests := []struct {
		w, h int
		want image.Rectangle
	}{
		{1280, 720, image.Rect(280, 0, 1000, 720)},
		{480, 360, image.Rect(105, 45, 375, 315)},
		{500, 500, image.Rect(0, 0, 500, 500)},
	}
	for _, tt := range tests {
		img := image.NewRGBA(image.Rect(0, 0, tt.w, tt.h))
		if got := squareCover(img).Bounds(); got != tt.want {
			t.Errorf("%dx%d: %v, ожидали %v", tt.w, tt.h, got, tt.want)
		}
	}
}
