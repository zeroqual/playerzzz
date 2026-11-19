package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Playlist struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Tracks []string `json:"tracks"`
	Cover  string   `json:"cover,omitempty"`
}

type Playlists struct {
	Items []Playlist `json:"playlists"`
}

var (
	ErrEmptyName = errors.New("empty playlist name")
	ErrNotFound  = errors.New("playlist not found")
)

func (s *Store) LoadPlaylists() (*Playlists, error) {
	p := &Playlists{}
	err := s.readJSON("playlists.json", p)
	return p, err
}

func (s *Store) SavePlaylists(p *Playlists) error {
	return s.writeJSON("playlists.json", p)
}

func (p *Playlists) Get(id string) (*Playlist, bool) {
	for i := range p.Items {
		if p.Items[i].ID == id {
			return &p.Items[i], true
		}
	}
	return nil, false
}

func (p *Playlists) Create(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	id := newID()
	p.Items = append(p.Items, Playlist{ID: id, Name: name, Tracks: []string{}})
	return id, nil
}

func (p *Playlists) Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	pl, ok := p.Get(id)
	if !ok {
		return ErrNotFound
	}
	pl.Name = name
	return nil
}

func (p *Playlists) Delete(id string) {
	p.Items = slices.DeleteFunc(p.Items, func(pl Playlist) bool { return pl.ID == id })
}

func (p *Playlists) RemoveTrack(path string) bool {
	changed := false
	for i := range p.Items {
		if p.Items[i].Contains(path) {
			p.Items[i].Remove(path)
			changed = true
		}
	}
	return changed
}

func (p *Playlists) Prune(gone func(path string) bool) bool {
	changed := false
	for i := range p.Items {
		pl := &p.Items[i]
		n := len(pl.Tracks)
		pl.Tracks = slices.DeleteFunc(pl.Tracks, gone)
		changed = changed || len(pl.Tracks) != n
	}
	return changed
}

func (p *Playlists) UniqueName(name string) string {
	taken := func(n string) bool {
		return slices.ContainsFunc(p.Items, func(pl Playlist) bool { return strings.EqualFold(pl.Name, n) })
	}
	if !taken(name) {
		return name
	}
	for i := 2; ; i++ {
		if n := fmt.Sprintf("%s (%d)", name, i); !taken(n) {
			return n
		}
	}
}

func (pl *Playlist) Contains(path string) bool { return slices.Contains(pl.Tracks, path) }

func (pl *Playlist) Add(path string) {
	if !pl.Contains(path) {
		pl.Tracks = append(pl.Tracks, path)
	}
}

func (pl *Playlist) Remove(path string) {
	pl.Tracks = slices.DeleteFunc(pl.Tracks, func(t string) bool { return t == path })
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
