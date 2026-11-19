package library

import (
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeroqual/playerzzz/internal/id3"
)

type Track struct {
	Path     string
	Title    string
	Artist   string
	HasCover bool
	YouTube  string
}

func Scan(dir string) ([]Track, error) {
	var tracks []Track

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() && path != dir {
				return fs.SkipDir
			}
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mp3") {
			return nil
		}
		tracks = append(tracks, Load(path))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(tracks, func(i, j int) bool {
		return strings.ToLower(tracks[i].Title) < strings.ToLower(tracks[j].Title)
	})
	return tracks, nil
}

func Load(path string) Track {
	name := filepath.Base(path)
	t := Track{Path: path, Title: strings.TrimSuffix(name, filepath.Ext(name))}
	if tag, err := id3.ReadFile(path); err == nil {
		if tag.Title != "" {
			t.Title = tag.Title
		}
		t.Artist = tag.Artist
		t.HasCover = len(tag.Cover) > 0
		t.YouTube = YouTubeID(tag.Source)
	}
	return t
}

func YouTubeID(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	var id string
	switch strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") {
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		id = u.Query().Get("v")
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	}
	if !validVideoID(id) {
		return ""
	}
	return id
}

func validVideoID(id string) bool {
	if len(id) != 11 {
		return false
	}
	for _, c := range id {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			return false
		}
	}
	return true
}

func Cover(path string) []byte {
	tag, err := id3.ReadFile(path)
	if err != nil {
		return nil
	}
	return tag.Cover
}
