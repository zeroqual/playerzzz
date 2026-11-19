package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const MaxDuration = 5 * time.Minute

var (
	ErrBadURL     = errors.New("not a YouTube video link")
	ErrTooLong    = fmt.Errorf("the video is longer than %d minutes", int(MaxDuration.Minutes()))
	ErrNoDuration = errors.New("couldn't get the video length (live stream?)")
)

type Progress func(done, total int64)

func ValidateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ErrBadURL
	}
	switch strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") {
	case "youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be":
		return nil
	}
	return ErrBadURL
}

func Download(ctx context.Context, link, dir string, progress Progress) (string, error) {
	link = strings.TrimSpace(link)
	if err := ValidateURL(link); err != nil {
		return "", err
	}
	if progress == nil {
		progress = func(int64, int64) {}
	}

	path, err := downloadNative(ctx, link, dir, progress)
	if err == nil {
		return path, nil
	}

	if errors.Is(err, ErrTooLong) || errors.Is(err, ErrNoDuration) || ctx.Err() != nil {
		return "", err
	}
	if !ytdlpAvailable() {
		return "", err
	}
	path, fbErr := downloadYtDlp(ctx, link, dir)
	if fbErr != nil {
		return "", fmt.Errorf("%w (yt-dlp fallback failed too: %v)", err, fbErr)
	}
	return path, nil
}
