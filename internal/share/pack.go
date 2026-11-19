package share

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

const PackExt = ".pzz"

const (
	manifestName = "manifest.json"
	coverName    = "cover.png"

	maxTrackSize = 200 << 20
	maxCoverSize = 8 << 20
	maxManifest  = 4 << 20
)

var ErrBadPack = errors.New("share: это не файл плейлиста playerzzz")

type Manifest struct {
	Version int         `json:"version"`
	Name    string      `json:"name"`
	Cover   string      `json:"cover,omitempty"`
	Tracks  []PackTrack `json:"tracks"`
}

type PackTrack struct {
	Title   string `json:"title"`
	Artist  string `json:"artist,omitempty"`
	YouTube string `json:"youtube,omitempty"`
	File    string `json:"file,omitempty"`
}

type ExportTrack struct {
	Track
	Path    string
	Include bool
}

func WritePack(w io.Writer, name string, cover []byte, tracks []ExportTrack, progress func(done int64)) error {
	zw := zip.NewWriter(w)
	m := Manifest{Version: 1, Name: name}

	if len(cover) > 0 {
		m.Cover = coverName
		if err := writeEntry(zw, coverName, cover); err != nil {
			return err
		}
	}

	var done int64
	for i, t := range tracks {
		pt := PackTrack{Title: t.Title, Artist: t.Artist, YouTube: t.YouTube}
		if t.Include {
			pt.File = fmt.Sprintf("tracks/%03d.mp3", i+1)
			n, err := copyFileToZip(zw, pt.File, t.Path)
			if err != nil {
				return fmt.Errorf("%s: %w", t.Title, err)
			}
			done += n
			if progress != nil {
				progress(done)
			}
		}
		m.Tracks = append(m.Tracks, pt)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := writeEntry(zw, manifestName, data); err != nil {
		return err
	}
	return zw.Close()
}

func writeEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func copyFileToZip(zw *zip.Writer, name, src string) (int64, error) {
	f, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	if err != nil {
		return 0, err
	}
	return io.Copy(w, f)
}

type Pack struct {
	Manifest Manifest
	zr       *zip.Reader
	closer   io.Closer
}

func OpenPack(path string) (*Pack, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, ErrBadPack
	}
	p, err := readPack(&rc.Reader)
	if err != nil {
		_ = rc.Close()
		return nil, err
	}
	p.closer = rc
	return p, nil
}

func ReadPack(r io.ReaderAt, size int64) (*Pack, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, ErrBadPack
	}
	return readPack(zr)
}

func readPack(zr *zip.Reader) (*Pack, error) {
	data, err := readEntry(zr, manifestName, maxManifest)
	if err != nil {
		return nil, ErrBadPack
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil || m.Version != 1 {
		return nil, ErrBadPack
	}
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" || len([]rune(m.Name)) > maxName || len(m.Tracks) > maxTracks {
		return nil, ErrBadPack
	}

	for _, t := range m.Tracks {
		if t.File != "" && !validEntry(t.File) {
			return nil, ErrBadPack
		}
	}
	if m.Cover != "" && m.Cover != coverName {
		return nil, ErrBadPack
	}
	return &Pack{Manifest: m, zr: zr}, nil
}

func validEntry(name string) bool {
	return fs.ValidPath(name) && path.Dir(name) == "tracks" && strings.HasSuffix(name, ".mp3")
}

func readEntry(zr *zip.Reader, name string, limit int64) ([]byte, error) {
	f, err := zr.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s: слишком большой", name)
	}
	return data, nil
}

func (p *Pack) Close() error {
	if p.closer != nil {
		return p.closer.Close()
	}
	return nil
}

func (p *Pack) Cover() []byte {
	if p.Manifest.Cover == "" {
		return nil
	}
	data, err := readEntry(p.zr, p.Manifest.Cover, maxCoverSize)
	if err != nil {
		return nil
	}
	return data
}

func (p *Pack) Playlist() Playlist {
	pl := Playlist{Name: p.Manifest.Name}
	for _, t := range p.Manifest.Tracks {
		pl.Tracks = append(pl.Tracks, Track{Title: t.Title, Artist: t.Artist, YouTube: t.YouTube})
	}
	return pl
}

func (p *Pack) ExtractTrack(i int, dst string) error {
	if i < 0 || i >= len(p.Manifest.Tracks) || p.Manifest.Tracks[i].File == "" {
		return errors.New("share: в файле нет этого трека")
	}
	src, err := p.zr.Open(p.Manifest.Tracks[i].File)
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(src, maxTrackSize+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxTrackSize {
		err = errors.New("share: трек слишком большой")
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

func MatchPack(p *Pack, library []LocalTrack) []Match {
	matches := MatchTracks(p.Playlist(), library)
	for i := range matches {
		if matches[i].Status != Found && p.Manifest.Tracks[i].File != "" {
			matches[i].Status = Packed
		}
	}
	return matches
}
