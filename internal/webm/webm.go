package webm

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	idEBML         = 0x1A45DFA3
	idSegment      = 0x18538067
	idTracks       = 0x1654AE6B
	idTrackEntry   = 0xAE
	idTrackNumber  = 0xD7
	idTrackType    = 0x83
	idCodecID      = 0x86
	idCodecPrivate = 0x63A2
	idCodecDelay   = 0x56AA
	idAudio        = 0xE1
	idChannels     = 0x9F
	idSampling     = 0xB5
	idCluster      = 0x1F43B675
	idBlockGroup   = 0xA0
	idBlock        = 0xA1
	idSimpleBlock  = 0xA3

	trackTypeAudio = 2
)

const unknownSize = math.MaxUint64

const maxBlockSize = 16 << 20

var (
	ErrNotWebM      = errors.New("webm: это не WebM/Matroska")
	ErrNoAudioTrack = errors.New("webm: в файле нет аудиодорожки")
)

type Track struct {
	Number       uint64
	CodecID      string
	CodecPrivate []byte
	CodecDelay   uint64
	Channels     int
	SampleRate   float64
}

type Reader struct {
	r       *bufio.Reader
	track   Track
	pending [][]byte
}

func NewReader(r io.Reader) (*Reader, error) {
	wr := &Reader{r: bufio.NewReaderSize(r, 64<<10)}

	id, size, err := wr.readHeader()
	if err != nil {
		return nil, err
	}
	if id != idEBML {
		return nil, ErrNotWebM
	}
	if err := wr.skip(size); err != nil {
		return nil, err
	}

	for {
		id, size, err := wr.readHeader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, ErrNoAudioTrack
			}
			return nil, err
		}
		switch id {
		case idSegment:

		case idTracks:
			body, err := wr.readBody(size)
			if err != nil {
				return nil, err
			}
			t, err := parseTracks(body)
			if err != nil {
				return nil, err
			}
			wr.track = t
			return wr, nil
		case idCluster:

			return nil, ErrNoAudioTrack
		default:
			if err := wr.skip(size); err != nil {
				return nil, err
			}
		}
	}
}

func (wr *Reader) Track() Track { return wr.track }

func (wr *Reader) ReadPacket() ([]byte, error) {
	for len(wr.pending) == 0 {
		id, size, err := wr.readHeader()
		if err != nil {
			return nil, err
		}
		switch id {
		case idSegment, idCluster, idBlockGroup:

		case idSimpleBlock, idBlock:
			body, err := wr.readBody(size)
			if err != nil {
				return nil, err
			}
			frames, track, err := parseBlock(body)
			if err != nil {
				return nil, err
			}
			if track == wr.track.Number {
				wr.pending = frames
			}
		default:
			if err := wr.skip(size); err != nil {
				return nil, err
			}
		}
	}
	p := wr.pending[0]
	wr.pending = wr.pending[1:]
	return p, nil
}

func (wr *Reader) readHeader() (id uint32, size uint64, err error) {
	v, n, err := readVint(wr.r, true)
	if err != nil {
		return 0, 0, err
	}
	if n > 4 {
		return 0, 0, fmt.Errorf("webm: слишком длинный ID элемента (%d байт)", n)
	}
	size, _, err = readVint(wr.r, false)
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	return uint32(v), size, err
}

func (wr *Reader) readBody(size uint64) ([]byte, error) {
	if size == unknownSize || size > maxBlockSize {
		return nil, fmt.Errorf("webm: неподдерживаемый размер элемента %d", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(wr.r, buf); err != nil {
		return nil, noEOF(err)
	}
	return buf, nil
}

func (wr *Reader) skip(size uint64) error {
	if size == unknownSize {
		return errors.New("webm: нельзя пропустить элемент неизвестного размера")
	}
	_, err := wr.r.Discard(int(size))
	return noEOF(err)
}

func readVint(r io.ByteReader, keepMarker bool) (uint64, int, error) {
	first, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	n := 1
	mask := byte(0x80)
	for n <= 8 && first&mask == 0 {
		n++
		mask >>= 1
	}
	if n > 8 {
		return 0, 0, errors.New("webm: битое число переменной длины")
	}

	v := uint64(first)
	if !keepMarker {
		v = uint64(first &^ mask)
	}
	allOnes := v == uint64(mask-1)
	for i := 1; i < n; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, 0, noEOF(err)
		}
		v = v<<8 | uint64(b)
		allOnes = allOnes && b == 0xFF
	}
	if !keepMarker && allOnes {
		return unknownSize, n, nil
	}
	return v, n, nil
}

type child struct {
	id   uint32
	body []byte
}

func children(data []byte) ([]child, error) {
	var out []child
	r := bytes.NewReader(data)
	for r.Len() > 0 {
		id, _, err := readVint(r, true)
		if err != nil {
			return nil, noEOF(err)
		}
		size, _, err := readVint(r, false)
		if err != nil {
			return nil, noEOF(err)
		}
		if size > uint64(r.Len()) {
			return nil, io.ErrUnexpectedEOF
		}
		start := len(data) - r.Len()
		out = append(out, child{id: uint32(id), body: data[start : start+int(size)]})
		_, _ = r.Seek(int64(size), io.SeekCurrent)
	}
	return out, nil
}

func readUint(b []byte) uint64 {
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

func readFloat(b []byte) float64 {
	switch len(b) {
	case 4:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b)))
	case 8:
		return math.Float64frombits(binary.BigEndian.Uint64(b))
	}
	return 0
}

func parseTracks(body []byte) (Track, error) {
	entries, err := children(body)
	if err != nil {
		return Track{}, err
	}
	for _, e := range entries {
		if e.id != idTrackEntry {
			continue
		}
		fields, err := children(e.body)
		if err != nil {
			return Track{}, err
		}
		var t Track
		var typ uint64
		for _, f := range fields {
			switch f.id {
			case idTrackNumber:
				t.Number = readUint(f.body)
			case idTrackType:
				typ = readUint(f.body)
			case idCodecID:
				t.CodecID = string(bytes.TrimRight(f.body, "\x00"))
			case idCodecPrivate:
				t.CodecPrivate = f.body
			case idCodecDelay:
				t.CodecDelay = readUint(f.body)
			case idAudio:
				audio, err := children(f.body)
				if err != nil {
					return Track{}, err
				}
				for _, a := range audio {
					switch a.id {
					case idChannels:
						t.Channels = int(readUint(a.body))
					case idSampling:
						t.SampleRate = readFloat(a.body)
					}
				}
			}
		}
		if typ == trackTypeAudio {
			if t.Channels == 0 {
				t.Channels = 1
			}
			return t, nil
		}
	}
	return Track{}, ErrNoAudioTrack
}

func parseBlock(b []byte) (frames [][]byte, track uint64, err error) {
	r := bytes.NewReader(b)
	track, _, err = readVint(r, false)
	if err != nil {
		return nil, 0, noEOF(err)
	}
	if r.Len() < 3 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	_, _ = r.Seek(2, io.SeekCurrent)
	flags, _ := r.ReadByte()
	data := b[len(b)-r.Len():]

	lacing := (flags >> 1) & 0x03
	if lacing == 0 {
		return [][]byte{data}, track, nil
	}

	if len(data) < 1 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	count := int(data[0]) + 1
	data = data[1:]
	sizes := make([]int, count)

	switch lacing {
	case 1:
		for i := 0; i < count-1; i++ {
			for {
				if len(data) == 0 {
					return nil, 0, io.ErrUnexpectedEOF
				}
				c := data[0]
				data = data[1:]
				sizes[i] += int(c)
				if c != 0xFF {
					break
				}
			}
		}
	case 3:
		r := bytes.NewReader(data)
		first, _, err := readVint(r, false)
		if err != nil {
			return nil, 0, noEOF(err)
		}
		sizes[0] = int(first)
		for i := 1; i < count-1; i++ {
			raw, n, err := readVint(r, false)
			if err != nil {
				return nil, 0, noEOF(err)
			}
			bias := int64(1)<<(7*n-1) - 1
			sizes[i] = sizes[i-1] + int(int64(raw)-bias)
		}
		data = data[len(data)-r.Len():]
	case 2:
		if len(data)%count != 0 {
			return nil, 0, errors.New("webm: битый блок с фиксированным лейсингом")
		}
		for i := range sizes {
			sizes[i] = len(data) / count
		}
	}

	if lacing != 2 {
		used := 0
		for _, s := range sizes[:count-1] {
			if s < 0 {
				return nil, 0, errors.New("webm: отрицательный размер кадра")
			}
			used += s
		}
		sizes[count-1] = len(data) - used
		if sizes[count-1] < 0 {
			return nil, 0, io.ErrUnexpectedEOF
		}
	}

	frames = make([][]byte, count)
	for i, s := range sizes {
		frames[i] = data[:s]
		data = data[s:]
	}
	return frames, track, nil
}

func noEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
