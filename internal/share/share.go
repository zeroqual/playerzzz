package share

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode"
)

const prefix = "pzz1:"

const (
	maxDecoded = 1 << 20
	maxTracks  = 2000
	maxName    = 200
)

var ErrBadCode = errors.New("share: это не код плейлиста playerzzz")

type Playlist struct {
	Name   string  `json:"n"`
	Tracks []Track `json:"t"`
}

type Track struct {
	Title   string `json:"t"`
	Artist  string `json:"a,omitempty"`
	YouTube string `json:"y,omitempty"`
}

func Encode(p Playlist) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func Decode(code string) (Playlist, error) {
	code = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, code)
	if !strings.HasPrefix(code, prefix) {
		return Playlist{}, ErrBadCode
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, prefix))
	if err != nil {
		return Playlist{}, ErrBadCode
	}
	data, err := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(raw)), maxDecoded+1))
	if err != nil || len(data) > maxDecoded {
		return Playlist{}, ErrBadCode
	}
	var p Playlist
	if err := json.Unmarshal(data, &p); err != nil {
		return Playlist{}, ErrBadCode
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len([]rune(p.Name)) > maxName || len(p.Tracks) > maxTracks {
		return Playlist{}, ErrBadCode
	}
	return p, nil
}

type LocalTrack struct {
	Path, Title, Artist, YouTube string
}

type Status int

const (
	Found Status = iota
	Download
	Missing
	Packed
)

type Match struct {
	Track  Track
	Status Status
	Path   string
}

func MatchTracks(p Playlist, library []LocalTrack) []Match {
	byYT := map[string]string{}
	byName := map[string]string{}
	for _, t := range library {
		if t.YouTube != "" {
			byYT[t.YouTube] = t.Path
		}
		k := key(t.Title, t.Artist)
		if _, dup := byName[k]; !dup {
			byName[k] = t.Path
		}
	}

	out := make([]Match, len(p.Tracks))
	for i, t := range p.Tracks {
		m := Match{Track: t, Status: Missing}
		if path, ok := byYT[t.YouTube]; ok && t.YouTube != "" {
			m.Status, m.Path = Found, path
		} else if path, ok := byName[key(t.Title, t.Artist)]; ok {
			m.Status, m.Path = Found, path
		} else if t.YouTube != "" {
			m.Status = Download
		}
		out[i] = m
	}
	return out
}

func key(title, artist string) string {
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	return norm(title) + "|" + norm(artist)
}
