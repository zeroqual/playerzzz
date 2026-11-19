package transcode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	shine "github.com/braheezy/shine-mp3/pkg/mp3"
	"github.com/tphakala/go-opus/opus"

	"github.com/zeroqual/playerzzz/internal/webm"
)

const sampleRate = 48000

const maxFrameSamples = 5760

func OpusWebMToMP3(ctx context.Context, in io.Reader, out io.Writer) error {
	demux, err := webm.NewReader(in)
	if err != nil {
		return err
	}
	track := demux.Track()
	if track.CodecID != "A_OPUS" {
		return fmt.Errorf("transcode: ожидался Opus, а в файле %q", track.CodecID)
	}
	if track.Channels < 1 || track.Channels > 2 {
		return fmt.Errorf("transcode: поддерживается моно и стерео, а каналов %d", track.Channels)
	}
	preSkip := opusPreSkip(track.CodecPrivate)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		once     sync.Once
		firstErr error
	)
	fail := func(err error) {
		once.Do(func() { firstErr = err; cancel() })
	}

	packets := make(chan []byte, 64)
	pcm := make(chan []int16, 16)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(packets)
		for {
			p, err := demux.ReadPacket()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				fail(err)
				return
			}
			select {
			case packets <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(pcm)
		dec, err := opus.NewDecoder(sampleRate, track.Channels)
		if err != nil {
			fail(err)
			return
		}
		skip := preSkip
		for p := range packets {
			buf := make([]int16, maxFrameSamples*track.Channels)
			n, err := dec.Decode(p, buf)
			if err != nil {
				fail(fmt.Errorf("transcode: декодирование opus: %w", err))
				return
			}
			samples := buf[:n*track.Channels]
			if skip > 0 {
				drop := min(skip, n)
				samples = samples[drop*track.Channels:]
				skip -= drop
			}
			if len(samples) == 0 {
				continue
			}
			select {
			case pcm <- samples:
			case <-ctx.Done():
				return
			}
		}
	}()

	encodeErr := encodeMP3(pcm, out, track.Channels)
	if encodeErr != nil {
		fail(encodeErr)
	}

	for range pcm {
	}
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

func encodeMP3(pcm <-chan []int16, out io.Writer, channels int) error {
	enc := shine.NewEncoder(sampleRate, channels)
	frame := int(enc.Mpeg.GranulesPerFrame) * shine.GRANULE_SIZE * channels

	buf := make([]int16, 0, frame*2)
	write := func(chunk []int16) error {
		data, n := enc.EncodeBufferInterleaved(chunk)
		_, err := out.Write(data[:n])
		return err
	}

	for samples := range pcm {
		buf = append(buf, samples...)
		for len(buf) >= frame {
			if err := write(buf[:frame]); err != nil {
				return err
			}
			buf = append(buf[:0], buf[frame:]...)
		}
	}
	if len(buf) > 0 {
		return write(buf)
	}
	return nil
}

func opusPreSkip(head []byte) int {
	if len(head) < 12 || string(head[:8]) != "OpusHead" {
		return 0
	}
	return int(binary.LittleEndian.Uint16(head[10:12]))
}
