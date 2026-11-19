package youtube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func ytdlpAvailable() bool {
	_, err1 := exec.LookPath("yt-dlp")
	_, err2 := exec.LookPath("ffmpeg")
	return err1 == nil && err2 == nil
}

func downloadYtDlp(ctx context.Context, link, dir string) (string, error) {
	out, err := runYtDlp(ctx,
		"--no-playlist",

		"--match-filter", fmt.Sprintf("duration <= %d", int(MaxDuration.Seconds())),
		"-x", "--audio-format", "mp3", "--audio-quality", "0",

		"--embed-thumbnail", "--convert-thumbnails", "jpg", "--embed-metadata",
		"-P", dir,
		"-o", "%(title)s.%(ext)s",

		"--print", "after_move:filepath",
		link,
	)
	if err != nil {
		return "", err
	}
	path := lastLine(out)
	if path == "" {
		return "", ErrTooLong
	}
	return path, nil
}

func runYtDlp(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	hideConsole(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("yt-dlp не успел за отведённое время")
		}
		if msg := lastLine(stderr.String()); msg != "" {
			return "", fmt.Errorf("yt-dlp: %s", msg)
		}
		return "", fmt.Errorf("yt-dlp: %w", err)
	}
	return stdout.String(), nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
