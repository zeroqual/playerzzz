package share

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPackRoundTrip(t *testing.T) {
	dir := t.TempDir()
	song := filepath.Join(dir, "song.mp3")
	audio := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 1000)
	if err := os.WriteFile(song, audio, 0o644); err != nil {
		t.Fatal(err)
	}
	cover := []byte("\x89PNG fake cover")

	tracks := []ExportTrack{
		{Track: Track{Title: "Локальный трек", Artist: "Я"}, Path: song, Include: true},
		{Track: Track{Title: "С YouTube", YouTube: "dQw4w9WgXcQ"}},
	}
	var buf bytes.Buffer
	var progressed int64
	if err := WritePack(&buf, "Вечер", cover, tracks, func(n int64) { progressed = n }); err != nil {
		t.Fatal(err)
	}
	if progressed != int64(len(audio)) {
		t.Fatalf("прогресс %d, ожидали %d", progressed, len(audio))
	}

	p, err := ReadPack(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest.Name != "Вечер" || len(p.Manifest.Tracks) != 2 || !bytes.Equal(p.Cover(), cover) {
		t.Fatalf("манифест: %+v", p.Manifest)
	}

	lib := []LocalTrack{{Path: "/lib/yt.mp3", Title: "что-то", YouTube: "dQw4w9WgXcQ"}}
	m := MatchPack(p, lib)
	if m[0].Status != Packed || m[1].Status != Found || m[1].Path != "/lib/yt.mp3" {
		t.Fatalf("сопоставление: %+v", m)
	}

	dst := filepath.Join(dir, "out.mp3")
	if err := p.ExtractTrack(0, dst); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, audio) {
		t.Fatal("распакованный трек не совпал")
	}
	if err := p.ExtractTrack(1, dst); err == nil {
		t.Fatal("трека-ссылки нет в файле — должна быть ошибка")
	}
}

func TestPackRejectsBadFiles(t *testing.T) {
	makeZip := func(manifest string) []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create("manifest.json")
		_, _ = w.Write([]byte(manifest))
		_ = zw.Close()
		return buf.Bytes()
	}
	cases := map[string][]byte{
		"не zip":           []byte("hello"),
		"битый json":       makeZip("{"),
		"чужая версия":     makeZip(`{"version":2,"name":"x","tracks":[]}`),
		"пустое имя":       makeZip(`{"version":1,"name":" ","tracks":[]}`),
		"выход из папки":   makeZip(`{"version":1,"name":"x","tracks":[{"title":"a","file":"../../evil.mp3"}]}`),
		"не та папка":      makeZip(`{"version":1,"name":"x","tracks":[{"title":"a","file":"other/1.mp3"}]}`),
		"странная обложка": makeZip(`{"version":1,"name":"x","cover":"../c.png","tracks":[]}`),
	}
	for name, data := range cases {
		if _, err := ReadPack(bytes.NewReader(data), int64(len(data))); !errors.Is(err, ErrBadPack) {
			t.Errorf("%s: ожидали ErrBadPack, получили %v", name, err)
		}
	}
}
