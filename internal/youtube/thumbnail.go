package youtube

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"time"
)

var thumbURLs = []string{
	"https://i.ytimg.com/vi/%s/maxresdefault.jpg",
	"https://i.ytimg.com/vi/%s/hqdefault.jpg",
}

func fetchCover(ctx context.Context, videoID string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	for _, u := range thumbURLs {
		img, err := getJPEG(ctx, client, fmt.Sprintf(u, videoID))
		if err != nil {
			lastErr = err
			continue
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, squareCover(img), &jpeg.Options{Quality: 90}); err != nil {
			return nil, err
		}
		return out.Bytes(), nil
	}
	return nil, lastErr
}

func getJPEG(ctx context.Context, c *http.Client, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("превью: HTTP %d", resp.StatusCode)
	}
	return jpeg.Decode(io.LimitReader(resp.Body, 8<<20))
}

func squareCover(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	if w*3 == h*4 {
		band := w * 9 / 16
		top := b.Min.Y + (h-band)/2
		b = image.Rect(b.Min.X, top, b.Max.X, top+band)
		w, h = b.Dx(), b.Dy()
	}

	side := min(w, h)
	x := b.Min.X + (w-side)/2
	y := b.Min.Y + (h-side)/2
	square := image.Rect(x, y, x+side, y+side)

	if sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(square)
	}
	return img
}
