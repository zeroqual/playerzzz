package id3

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode/utf16"
)

func Encode(t Tag) []byte {
	var frames bytes.Buffer
	if t.Title != "" {
		writeFrame(&frames, "TIT2", textFrame(t.Title))
	}
	if t.Artist != "" {
		writeFrame(&frames, "TPE1", textFrame(t.Artist))
	}
	if t.Source != "" {
		writeFrame(&frames, "WOAS", []byte(t.Source))
	}
	if len(t.Cover) > 0 {
		mime := t.Mime
		if mime == "" {
			mime = normMime("image/jpeg", t.Cover)
		}
		var b bytes.Buffer
		b.WriteByte(0)
		b.WriteString(mime)
		b.WriteByte(0)
		b.WriteByte(3)
		b.WriteByte(0)
		b.Write(t.Cover)
		writeFrame(&frames, "APIC", b.Bytes())
	}

	var out bytes.Buffer
	out.WriteString("ID3")
	out.Write([]byte{3, 0, 0})
	out.Write(toSyncsafe(uint32(frames.Len())))
	out.Write(frames.Bytes())
	return out.Bytes()
}

func WriteWithAudio(w io.Writer, t Tag, audio io.Reader) error {
	if _, err := w.Write(Encode(t)); err != nil {
		return err
	}
	_, err := io.Copy(w, audio)
	return err
}

func textFrame(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 1+2+2*len(u))
	b[0] = 1
	b[1], b[2] = 0xFF, 0xFE
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[3+2*i:], c)
	}
	return b
}

func writeFrame(w *bytes.Buffer, id string, data []byte) {
	w.WriteString(id)
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(data)))
	w.Write(size[:])
	w.Write([]byte{0, 0})
	w.Write(data)
}

func toSyncsafe(n uint32) []byte {
	return []byte{byte(n >> 21 & 0x7F), byte(n >> 14 & 0x7F), byte(n >> 7 & 0x7F), byte(n & 0x7F)}
}
