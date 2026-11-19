package transcode

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"

	gomp3 "github.com/hajimehoshi/go-mp3"
)

func TestOpusWebMToMP3(t *testing.T) {
	in, err := os.Open("testdata/tone.webm")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	var out bytes.Buffer
	if err := OpusWebMToMP3(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}

	dec, err := gomp3.NewDecoder(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("получился невалидный mp3: %v", err)
	}
	if dec.SampleRate() != sampleRate {
		t.Fatalf("частота %d, ожидали %d", dec.SampleRate(), sampleRate)
	}
	raw, err := io.ReadAll(dec)
	if err != nil {
		t.Fatal(err)
	}

	n := len(raw) / 4
	left, right := make([]float64, n), make([]float64, n)
	for i := range n {
		left[i] = float64(int16(binary.LittleEndian.Uint16(raw[i*4:])))
		right[i] = float64(int16(binary.LittleEndian.Uint16(raw[i*4+2:])))
	}

	if got := float64(n) / sampleRate; math.Abs(got-3) > 0.1 {
		t.Fatalf("длительность %.3f с, ожидали ~3 с", got)
	}

	mid := func(s []float64) []float64 { return s[n/4 : 3*n/4] }
	checkFreq(t, "левый", mid(left), 440)
	checkFreq(t, "правый", mid(right), 660)
}

func TestNotOpus(t *testing.T) {
	err := OpusWebMToMP3(context.Background(), bytes.NewReader([]byte("не webm")), io.Discard)
	if err == nil {
		t.Fatal("ожидали ошибку")
	}
}

func checkFreq(t *testing.T, name string, s []float64, want float64) {
	t.Helper()
	var rms float64
	crossings := 0
	for i := 1; i < len(s); i++ {
		rms += s[i] * s[i]
		if (s[i-1] < 0) != (s[i] < 0) {
			crossings++
		}
	}
	rms = math.Sqrt(rms / float64(len(s)))
	if rms < 1000 {
		t.Fatalf("%s канал почти тихий (RMS %.0f)", name, rms)
	}
	got := float64(crossings) / 2 / (float64(len(s)) / sampleRate)
	if math.Abs(got-want) > want*0.02 {
		t.Fatalf("%s канал: частота %.1f Гц, ожидали %.0f", name, got, want)
	}
}
