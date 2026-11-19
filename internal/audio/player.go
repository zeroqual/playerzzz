package audio

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

const outputRate beep.SampleRate = 44100

type Player struct {
	mu sync.Mutex

	stream beep.StreamSeekCloser
	format beep.Format
	ctrl   *beep.Ctrl
	vol    *effects.Volume

	volume float64
	gen    int

	OnEnd func()
}

func New() (*Player, error) {
	if err := speaker.Init(outputRate, outputRate.N(time.Second/10)); err != nil {
		return nil, fmt.Errorf("init speaker: %w", err)
	}
	return &Player{volume: 0.8}, nil
}

func (p *Player) Play(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	stream, format, err := mp3.Decode(f)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("decode %s: %w", path, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	speaker.Clear()
	if p.stream != nil {
		_ = p.stream.Close()
	}

	p.gen++
	gen := p.gen

	var s beep.Streamer = stream
	if format.SampleRate != outputRate {
		s = beep.Resample(4, format.SampleRate, outputRate, stream)
	}
	ctrl := &beep.Ctrl{Streamer: s}
	vol := &effects.Volume{Streamer: ctrl, Base: 2}
	applyVolume(vol, p.volume)

	p.stream, p.format, p.ctrl, p.vol = stream, format, ctrl, vol

	speaker.Play(beep.Seq(vol, beep.Callback(func() {
		go p.finished(gen)
	})))
	return nil
}

func (p *Player) finished(gen int) {
	p.mu.Lock()
	stale := gen != p.gen
	cb := p.OnEnd
	p.mu.Unlock()
	if !stale && cb != nil {
		cb()
	}
}

func (p *Player) TogglePause() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return true
	}
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	paused := p.ctrl.Paused
	speaker.Unlock()
	return paused
}

func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return false
	}
	speaker.Lock()
	defer speaker.Unlock()
	return !p.ctrl.Paused
}

func (p *Player) Progress() (pos, length time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == nil {
		return 0, 0
	}
	speaker.Lock()
	defer speaker.Unlock()
	return p.format.SampleRate.D(p.stream.Position()), p.format.SampleRate.D(p.stream.Len())
}

func (p *Player) Seek(d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == nil {
		return nil
	}
	speaker.Lock()
	defer speaker.Unlock()
	n := p.format.SampleRate.N(d)
	n = max(0, min(n, p.stream.Len()-1))
	return p.stream.Seek(n)
}

func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.volume = max(0, min(v, 1))
	if p.vol == nil {
		return
	}
	speaker.Lock()
	applyVolume(p.vol, p.volume)
	speaker.Unlock()
}

func applyVolume(vol *effects.Volume, v float64) {
	if v <= 0.001 {
		vol.Silent = true
		return
	}
	vol.Silent = false
	vol.Volume = math.Log2(v) * 2
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	speaker.Clear()
	p.gen++
	if p.stream != nil {
		_ = p.stream.Close()
	}
	p.stream, p.ctrl, p.vol = nil, nil, nil
}

func (p *Player) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	speaker.Clear()
	if p.stream != nil {
		_ = p.stream.Close()
		p.stream = nil
	}
}
