package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	xdraw "golang.org/x/image/draw"

	"github.com/zeroqual/playerzzz/internal/library"
)

type coverCache struct {
	mu      sync.Mutex
	loaded  map[string]fyne.Resource
	pending map[string]bool
	sem     chan struct{}
}

func newCoverCache() *coverCache {
	return &coverCache{
		loaded:  map[string]fyne.Resource{},
		pending: map[string]bool{},
		sem:     make(chan struct{}, 4),
	}
}

func (c *coverCache) get(t library.Track, onReady func()) fyne.Resource {
	if !t.HasCover {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if res, ok := c.loaded[t.Path]; ok {
		return res
	}
	if c.pending[t.Path] {
		return nil
	}
	c.pending[t.Path] = true
	go func() {
		c.sem <- struct{}{}
		data := library.Cover(t.Path)
		<-c.sem

		var res fyne.Resource
		if len(data) > 0 {
			res = fyne.NewStaticResource("cover:"+t.Path, data)
		}
		c.mu.Lock()
		c.loaded[t.Path] = res
		delete(c.pending, t.Path)
		c.mu.Unlock()
		if res != nil {
			fyne.Do(onReady)
		}
	}()
	return nil
}

func (c *coverCache) mosaic(covers []fyne.Resource) fyne.Resource {
	key := "mosaic"
	for _, r := range covers {
		key += "|" + r.Name()
	}
	c.mu.Lock()
	res, ok := c.loaded[key]
	c.mu.Unlock()
	if ok {
		return res
	}

	const half = 256
	dst := image.NewRGBA(image.Rect(0, 0, 2*half, 2*half))
	for i, r := range covers[:4] {
		src, _, err := image.Decode(bytes.NewReader(r.Content()))
		if err != nil {
			return nil
		}
		x, y := (i%2)*half, (i/2)*half
		xdraw.ApproxBiLinear.Scale(dst, image.Rect(x, y, x+half, y+half), src, squareRect(src.Bounds()), xdraw.Src, nil)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil
	}
	res = fyne.NewStaticResource(key+".png", out.Bytes())
	c.mu.Lock()
	c.loaded[key] = res
	c.mu.Unlock()
	return res
}

func squareRect(b image.Rectangle) image.Rectangle {
	side := min(b.Dx(), b.Dy())
	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	return image.Rect(x, y, x+side, y+side)
}

func (c *coverCache) forget(path string) {
	c.mu.Lock()
	delete(c.loaded, path)
	c.mu.Unlock()
}

const avatarFile = "avatar.png"

var errBadImage = errors.New("bad image")

func saveSquarePNG(data []byte, size int, path string) error {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return errBadImage
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, squareRect(src.Bounds()), xdraw.Src, nil)

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func saveAvatar(dir string, data []byte) error {
	return saveSquarePNG(data, 256, filepath.Join(dir, avatarFile))
}

const coversDir = "covers"

func (a *App) playlistCover(file string) fyne.Resource {
	if file == "" {
		return nil
	}
	key := "plcover:" + file
	a.covers.mu.Lock()
	res, ok := a.covers.loaded[key]
	a.covers.mu.Unlock()
	if ok {
		return res
	}
	data, err := os.ReadFile(filepath.Join(a.store.Dir, coversDir, file))
	if err != nil {
		return nil
	}

	res = fyne.NewStaticResource(file, data)
	a.covers.mu.Lock()
	a.covers.loaded[key] = res
	a.covers.mu.Unlock()
	return res
}

func (a *App) setPlaylistCover(id string, data []byte) error {
	pl, ok := a.pls.Get(id)
	if !ok {
		return nil
	}
	file := fmt.Sprintf("%s-%d.png", id, time.Now().UnixNano())
	if err := saveSquarePNG(data, 512, filepath.Join(a.store.Dir, coversDir, file)); err != nil {
		return err
	}
	a.removePlaylistCoverFile(pl.Cover)
	pl.Cover = file
	return nil
}

func (a *App) removePlaylistCoverFile(file string) {
	if file != "" {
		_ = os.Remove(filepath.Join(a.store.Dir, coversDir, file))
	}
}

func (a *App) avatarResource() fyne.Resource {
	if a.cfg.Profile.Avatar == "" {
		return nil
	}
	if a.avatarRes == nil {
		data, err := os.ReadFile(filepath.Join(a.store.Dir, a.cfg.Profile.Avatar))
		if err != nil {
			return nil
		}

		a.avatarVer++
		a.avatarRes = fyne.NewStaticResource("avatar-"+strconv.Itoa(a.avatarVer)+".png", data)
	}
	return a.avatarRes
}
