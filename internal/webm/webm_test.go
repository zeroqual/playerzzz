package webm

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

const toneFile = "../transcode/testdata/tone.webm"

func TestReaderTone(t *testing.T) {
	f, err := os.Open(toneFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := r.Track()
	if tr.CodecID != "A_OPUS" || tr.Channels != 2 || tr.SampleRate != 48000 {
		t.Fatalf("дорожка: %+v", tr)
	}
	if !bytes.HasPrefix(tr.CodecPrivate, []byte("OpusHead")) {
		t.Fatalf("CodecPrivate не OpusHead: %q", tr.CodecPrivate)
	}

	packets := 0
	for {
		p, err := r.ReadPacket()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(p) == 0 {
			t.Fatal("пустой пакет")
		}
		packets++
	}

	if packets < 145 || packets > 160 {
		t.Fatalf("пакетов %d, ожидали ~151", packets)
	}
}

func TestReaderNotWebM(t *testing.T) {
	_, err := NewReader(bytes.NewReader([]byte("ID3 это mp3, а не webm")))
	if !errors.Is(err, ErrNotWebM) {
		t.Fatalf("ожидали ErrNotWebM, получили %v", err)
	}
}

func TestReadVint(t *testing.T) {
	tests := []struct {
		in     []byte
		marker bool
		want   uint64
		n      int
	}{
		{[]byte{0x81}, false, 1, 1},
		{[]byte{0x40, 0x02}, false, 2, 2},
		{[]byte{0x1A, 0x45, 0xDF, 0xA3}, true, idEBML, 4},
		{[]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, false, unknownSize, 8},
	}
	for _, tt := range tests {
		got, n, err := readVint(bytes.NewReader(tt.in), tt.marker)
		if err != nil || got != tt.want || n != tt.n {
			t.Errorf("readVint(%x) = %d, %d, %v; ожидали %d, %d", tt.in, got, n, err, tt.want, tt.n)
		}
	}
}

func TestParseBlockLacing(t *testing.T) {
	head := func(flags byte) []byte { return []byte{0x81, 0x00, 0x00, flags} }
	a, b, c := bytes.Repeat([]byte{1}, 300), []byte{2, 2}, []byte{3, 3, 3}

	tests := []struct {
		name  string
		block []byte
		want  [][]byte
	}{
		{"без лейсинга", append(head(0x80), a...), [][]byte{a}},
		{"xiph", cat(head(0x82), []byte{2, 0xFF, 45, 2}, a, b, c), [][]byte{a, b, c}},

		{"ebml", cat(head(0x86), []byte{2, 0x41, 0x2C}, ebmlSigned(-298), a, b, c), [][]byte{a, b, c}},
		{"фиксированный", cat(head(0x84), []byte{2}, c, c, c), [][]byte{c, c, c}},
	}
	for _, tt := range tests {
		frames, track, err := parseBlock(tt.block)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if track != 1 || len(frames) != len(tt.want) {
			t.Fatalf("%s: дорожка %d, кадров %d", tt.name, track, len(frames))
		}
		for i := range frames {
			if !bytes.Equal(frames[i], tt.want[i]) {
				t.Errorf("%s: кадр %d не совпал", tt.name, i)
			}
		}
	}
}

func cat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

func ebmlSigned(d int) []byte {
	v := uint16(d + (1<<13 - 1))
	return []byte{byte(v>>8) | 0x40, byte(v)}
}
