package share

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := Playlist{Name: "Вечер", Tracks: []Track{
		{Title: "Neon Avenue", Artist: "Night Drive", YouTube: "dQw4w9WgXcQ"},
		{Title: "Тёплый дождь", Artist: "Sora Waves"},
	}}
	code, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "pzz1:") {
		t.Fatalf("код без префикса: %q", code)
	}

	mangled := code[:20] + "\n  " + code[20:] + " \n"
	out, err := Decode(mangled)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || len(out.Tracks) != 2 || out.Tracks[0] != in.Tracks[0] || out.Tracks[1] != in.Tracks[1] {
		t.Fatalf("got %+v", out)
	}
}

func TestCodeIsShort(t *testing.T) {
	p := Playlist{Name: "Большой"}
	for i := 0; i < 50; i++ {
		p.Tracks = append(p.Tracks, Track{Title: "Track title number", Artist: "Some Artist", YouTube: "dQw4w9WgXcQ"})
	}
	code, _ := Encode(p)
	if len(code) > 1500 {
		t.Fatalf("код для 50 треков слишком длинный: %d символов", len(code))
	}
}

func TestBadCodes(t *testing.T) {
	bomb := func() string {
		var buf bytes.Buffer
		w, _ := flate.NewWriter(&buf, flate.BestCompression)
		_, _ = w.Write(bytes.Repeat([]byte("a"), 4<<20))
		_ = w.Close()
		return "pzz1:" + base64.RawURLEncoding.EncodeToString(buf.Bytes())
	}()
	empty, _ := Encode(Playlist{Name: "  "})

	for name, code := range map[string]string{
		"пусто":         "",
		"без префикса":  "hello",
		"не base64":     "pzz1:!!!",
		"не deflate":    "pzz1:aGVsbG8",
		"zip-бомба":     bomb,
		"пустое имя":    empty,
		"чужой префикс": "pzz2:aGVsbG8",
	} {
		if _, err := Decode(code); !errors.Is(err, ErrBadCode) {
			t.Errorf("%s: ожидали ErrBadCode, получили %v", name, err)
		}
	}
}

func TestMatchTracks(t *testing.T) {
	lib := []LocalTrack{
		{Path: "/m/neon.mp3", Title: "Neon Avenue", Artist: "Night Drive", YouTube: "aaaaaaaaaaa"},
		{Path: "/m/rain.mp3", Title: "Тёплый дождь!", Artist: "Sora  Waves"},
	}
	p := Playlist{Name: "x", Tracks: []Track{
		{Title: "другое название", YouTube: "aaaaaaaaaaa"},
		{Title: "тёплый дождь", Artist: "sora waves"},
		{Title: "New Song", Artist: "Band", YouTube: "bbbbbbbbbbb"},
		{Title: "Local only", Artist: "Someone"},
	}}
	m := MatchTracks(p, lib)
	want := []struct {
		s    Status
		path string
	}{{Found, "/m/neon.mp3"}, {Found, "/m/rain.mp3"}, {Download, ""}, {Missing, ""}}
	for i, w := range want {
		if m[i].Status != w.s || m[i].Path != w.path {
			t.Errorf("%d: %+v, ожидали %v %q", i, m[i], w.s, w.path)
		}
	}
}
