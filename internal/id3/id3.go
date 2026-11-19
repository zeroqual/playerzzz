package id3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

type Tag struct {
	Title  string
	Artist string
	Cover  []byte
	Mime   string
	Source string
}

var ErrNoTag = errors.New("id3: нет тега ID3v2")

const maxTagSize = 64 << 20

func ReadFile(path string) (Tag, error) {
	f, err := os.Open(path)
	if err != nil {
		return Tag{}, err
	}
	defer f.Close()
	return Read(f)
}

func Read(r io.Reader) (Tag, error) {
	var hdr [10]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Tag{}, ErrNoTag
		}
		return Tag{}, err
	}
	if string(hdr[:3]) != "ID3" {
		return Tag{}, ErrNoTag
	}
	ver, flags := hdr[3], hdr[5]
	if ver < 2 || ver > 4 {
		return Tag{}, errors.New("id3: неизвестная версия")
	}
	size := syncsafe(hdr[6:10])
	if size > maxTagSize {
		return Tag{}, errors.New("id3: тег слишком большой")
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		return Tag{}, err
	}

	if flags&0x80 != 0 && ver < 4 {
		body = deunsync(body)
	}

	if flags&0x40 != 0 && len(body) >= 4 {
		var ext int
		if ver == 4 {
			ext = int(syncsafe(body[:4]))
		} else {
			ext = int(binary.BigEndian.Uint32(body[:4])) + 4
		}
		if ext > len(body) {
			return Tag{}, errors.New("id3: битый расширенный заголовок")
		}
		body = body[ext:]
	}

	var t Tag
	for len(body) > 0 {
		id, data, rest, ok := nextFrame(body, ver)
		if !ok {
			break
		}
		body = rest
		switch id {
		case "TIT2", "TT2":
			t.Title = decodeText(data)
		case "TPE1", "TP1":
			t.Artist = decodeText(data)
		case "WOAS", "WAS":

			if i := bytes.IndexByte(data, 0); i >= 0 {
				data = data[:i]
			}
			t.Source = strings.TrimSpace(string(data))
		case "APIC":
			if t.Cover == nil {
				t.Mime, t.Cover = decodeAPIC(data)
			}
		case "PIC":
			if t.Cover == nil {
				t.Mime, t.Cover = decodePIC(data)
			}
		}
	}
	return t, nil
}

func nextFrame(b []byte, ver byte) (id string, data, rest []byte, ok bool) {
	if ver == 2 {
		if len(b) < 6 || b[0] == 0 {
			return "", nil, nil, false
		}
		size := int(b[3])<<16 | int(b[4])<<8 | int(b[5])
		if 6+size > len(b) {
			return "", nil, nil, false
		}
		return string(b[:3]), b[6 : 6+size], b[6+size:], true
	}

	if len(b) < 10 || b[0] == 0 {
		return "", nil, nil, false
	}
	var size int
	if ver == 4 {
		size = int(syncsafe(b[4:8]))
	} else {
		size = int(binary.BigEndian.Uint32(b[4:8]))
	}
	if size < 0 || 10+size > len(b) {
		return "", nil, nil, false
	}
	data = b[10 : 10+size]
	if ver == 4 {
		fl := b[9]
		if fl&0x01 != 0 && len(data) >= 4 {
			data = data[4:]
		}
		if fl&0x02 != 0 {
			data = deunsync(data)
		}
	}
	return string(b[:4]), data, b[10+size:], true
}

func decodeText(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	s := decodeString(b[0], b[1:])

	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func decodeAPIC(b []byte) (mime string, data []byte) {
	if len(b) < 4 {
		return "", nil
	}
	enc := b[0]
	end := bytes.IndexByte(b[1:], 0)
	if end < 0 {
		return "", nil
	}
	mime = strings.ToLower(string(b[1 : 1+end]))
	rest := b[1+end+1:]
	if len(rest) < 1 {
		return "", nil
	}
	rest = rest[1:]
	rest = skipString(enc, rest)
	return normMime(mime, rest), rest
}

func decodePIC(b []byte) (mime string, data []byte) {
	if len(b) < 6 {
		return "", nil
	}
	enc, format := b[0], strings.ToLower(string(b[1:4]))
	rest := skipString(enc, b[5:])
	if format == "png" {
		return "image/png", rest
	}
	return normMime("image/jpeg", rest), rest
}

func normMime(mime string, data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG")):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		return "image/jpeg"
	case mime == "image/jpg":
		return "image/jpeg"
	}
	return mime
}

func skipString(enc byte, b []byte) []byte {
	if enc == 1 || enc == 2 {
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return b[i+2:]
			}
		}
		return nil
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return b[i+1:]
	}
	return nil
}

func decodeString(enc byte, b []byte) string {
	switch enc {
	case 0:
		r := make([]rune, len(b))
		for i, c := range b {
			r[i] = rune(c)
		}
		return string(r)
	case 1, 2:
		bigEndian := enc == 2
		if len(b) >= 2 {
			switch {
			case b[0] == 0xFF && b[1] == 0xFE:
				bigEndian, b = false, b[2:]
			case b[0] == 0xFE && b[1] == 0xFF:
				bigEndian, b = true, b[2:]
			}
		}
		u := make([]uint16, len(b)/2)
		for i := range u {
			if bigEndian {
				u[i] = binary.BigEndian.Uint16(b[2*i:])
			} else {
				u[i] = binary.LittleEndian.Uint16(b[2*i:])
			}
		}
		return string(utf16.Decode(u))
	default:
		return string(b)
	}
}

func syncsafe(b []byte) uint32 {
	return uint32(b[0]&0x7F)<<21 | uint32(b[1]&0x7F)<<14 | uint32(b[2]&0x7F)<<7 | uint32(b[3]&0x7F)
}

func deunsync(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		out = append(out, b[i])
		if b[i] == 0xFF && i+1 < len(b) && b[i+1] == 0x00 {
			i++
		}
	}
	return out
}
