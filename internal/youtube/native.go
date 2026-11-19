package youtube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	yt "github.com/kkdai/youtube/v2"

	"github.com/zeroqual/playerzzz/internal/fsutil"
	"github.com/zeroqual/playerzzz/internal/id3"
	"github.com/zeroqual/playerzzz/internal/transcode"
)

var (
	errNoOpus = errors.New("the video has no Opus audio track")

	ErrBlocked = errors.New("YouTube blocked the download (403); try later or install yt-dlp as a fallback")
)

type attempt struct {
	name   string
	client yt.ClientInfo
	params string
}

var attempts = []attempt{
	{"android+2AMB", yt.AndroidClient, "2AMB"},
	{"android", yt.AndroidClient, ""},
	{"ios+2AMB", yt.IOSClient, "2AMB"},
	{"ios", yt.IOSClient, ""},
	{"web", yt.WebClient, ""},
}

var clientMu sync.Mutex

func downloadNative(ctx context.Context, link, dir string, progress Progress) (string, error) {
	clientMu.Lock()
	defer clientMu.Unlock()
	saved := yt.DefaultClient
	defer func() { yt.DefaultClient = saved }()

	var errs []error
	for _, a := range attempts {
		yt.DefaultClient = a.client
		client := &yt.Client{PlayerParams: a.params}

		path, err := downloadWith(ctx, client, link, dir, progress)
		if err == nil {
			return path, nil
		}

		if errors.Is(err, ErrTooLong) || errors.Is(err, ErrNoDuration) || ctx.Err() != nil {
			return "", err
		}
		errs = append(errs, fmt.Errorf("%s: %w", a.name, err))
	}

	all := errors.Join(errs...)

	log.Printf("youtube: все попытки не удались:\n%v", all)
	if strings.Contains(all.Error(), "403") {
		return "", ErrBlocked
	}
	return "", all
}

func downloadWith(ctx context.Context, client *yt.Client, link, dir string, progress Progress) (string, error) {
	video, err := client.GetVideoContext(ctx, link)
	if err != nil {
		return "", err
	}
	if video.Duration <= 0 {
		return "", ErrNoDuration
	}
	if video.Duration > MaxDuration {
		return "", ErrTooLong
	}

	format, err := bestOpus(video.Formats)
	if err != nil {
		return "", err
	}

	stream, size, err := client.GetStreamContext(ctx, video, format)
	if err != nil {
		return "", fmt.Errorf("поток: %w", err)
	}
	defer stream.Close()

	coverCh := make(chan []byte, 1)
	go func() {
		cover, err := fetchCover(ctx, video.ID)
		if err != nil {
			log.Printf("youtube: превью не скачалось: %v", err)
		}
		coverCh <- cover
	}()

	tmp, err := os.CreateTemp(dir, ".download-*.part")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	progress(0, size)
	src := &countingReader{r: stream, total: size, progress: progress}
	if err := transcode.OpusWebMToMP3(ctx, src, tmp); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	tag := id3.Tag{
		Title:  video.Title,
		Artist: video.Author,
		Cover:  <-coverCh,

		Source: "https://www.youtube.com/watch?v=" + video.ID,
	}
	final := fsutil.UniquePath(dir, fsutil.SafeName(video.Title, "youtube-"+video.ID), ".mp3")
	if err := withTag(tmp.Name(), final, tag); err != nil {
		return "", err
	}
	return final, nil
}

func withTag(audioPath, final string, tag id3.Tag) error {
	audio, err := os.Open(audioPath)
	if err != nil {
		return err
	}
	defer audio.Close()

	out, err := os.CreateTemp(filepath.Dir(final), ".tagged-*.part")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	if err := id3.WriteWithAudio(out, tag, audio); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), final)
}

func bestOpus(formats yt.FormatList) (*yt.Format, error) {
	opus := formats.Select(func(f yt.Format) bool {
		return strings.HasPrefix(f.MimeType, "audio/webm") && strings.Contains(f.MimeType, "opus")
	})
	if len(opus) == 0 {
		return nil, errNoOpus
	}
	sort.Slice(opus, func(i, j int) bool { return opus[i].Bitrate > opus[j].Bitrate })
	return &opus[0], nil
}

type countingReader struct {
	r        io.Reader
	done     int64
	total    int64
	progress Progress
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.done += int64(n)
	c.progress(c.done, c.total)
	return n, err
}
