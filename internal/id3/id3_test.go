package id3

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	gomp3 "github.com/hajimehoshi/go-mp3"
)

func TestReadFFmpegTags(t *testing.T) {
	for _, name := range []string{"testdata/tagged_v23.mp3", "testdata/tagged_v24.mp3"} {
		tag, err := ReadFile(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if tag.Title != "Тёплый дождь" || tag.Artist != "Sora Waves" {
			t.Errorf("%s: title=%q artist=%q", name, tag.Title, tag.Artist)
		}
		if tag.Mime != "image/png" || !bytes.HasPrefix(tag.Cover, []byte("\x89PNG")) {
			t.Errorf("%s: обложка %q, %d байт", name, tag.Mime, len(tag.Cover))
		}
	}
}

func TestRoundTrip(t *testing.T) {
	cover := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0xFF, 0x00, 7}, 100)...)
	in := Tag{Title: "Neon Avenue — ночная версия", Artist: "Night Drive", Cover: cover,
		Source: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}

	got, err := Read(bytes.NewReader(Encode(in)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != in.Title || got.Artist != in.Artist || got.Source != in.Source {
		t.Fatalf("got %q / %q / %q", got.Title, got.Artist, got.Source)
	}
	if got.Mime != "image/jpeg" || !bytes.Equal(got.Cover, cover) {
		t.Fatalf("обложка: %q, %d байт", got.Mime, len(got.Cover))
	}
}

func TestNoTag(t *testing.T) {
	for _, in := range [][]byte{nil, []byte("ID"), []byte("\xFF\xFB\x90\x00 просто mp3-кадр")} {
		if _, err := Read(bytes.NewReader(in)); !errors.Is(err, ErrNoTag) {
			t.Errorf("%q: ожидали ErrNoTag, получили %v", in, err)
		}
	}
}

func TestDecoderSkipsOurTag(t *testing.T) {
	audio, err := os.ReadFile("testdata/tagged_v24.mp3")
	if err != nil {
		t.Fatal(err)
	}

	size := int(syncsafe(audio[6:10])) + 10

	var buf bytes.Buffer
	if err := WriteWithAudio(&buf, Tag{Title: "x", Cover: []byte{0xFF, 0xD8}}, bytes.NewReader(audio[size:])); err != nil {
		t.Fatal(err)
	}
	dec, err := gomp3.NewDecoder(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("декодер не принял файл с нашим тегом: %v", err)
	}
	n, err := io.Copy(io.Discard, dec)
	if err != nil || n == 0 {
		t.Fatalf("декодировано %d байт, %v", n, err)
	}
}
