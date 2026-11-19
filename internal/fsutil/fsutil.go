package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func SafeName(title, fallback string) string {
	var b strings.Builder
	for _, r := range title {
		switch {
		case strings.ContainsRune(`<>:"/\|?*`, r), unicode.IsControl(r):
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	name := strings.Join(strings.Fields(b.String()), " ")
	name = strings.TrimRight(name, ". ")
	if r := []rune(name); len(r) > 120 {
		name = strings.TrimRight(string(r[:120]), ". ")
	}
	if name == "" {
		name = fallback
	}
	return name
}

func UniquePath(dir, name, ext string) string {
	p := filepath.Join(dir, name+ext)
	for i := 2; ; i++ {
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))
	}
}
