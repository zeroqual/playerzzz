package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeName(t *testing.T) {
	tests := map[string]string{
		`AC/DC: Back In Black?`: "AC DC Back In Black",
		"  много   пробелов  ":  "много пробелов",
		"точки в конце...":      "точки в конце",
		`<>:"/\|?*`:             "fallback",
	}
	for in, want := range tests {
		if got := SafeName(in, "fallback"); got != want {
			t.Errorf("SafeName(%q) = %q, ожидали %q", in, got, want)
		}
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	first := UniquePath(dir, "song", ".mp3")
	if err := os.WriteFile(first, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := UniquePath(dir, "song", ".mp3"), filepath.Join(dir, "song (2).mp3"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
