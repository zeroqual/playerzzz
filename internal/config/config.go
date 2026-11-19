package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	Language string  `json:"language"`
	Theme    string  `json:"theme"`
	Folder   string  `json:"folder"`
	Covers   bool    `json:"covers"`
	Volume   float64 `json:"volume"`
	Profile  Profile `json:"profile"`

	SystemFrame bool `json:"systemFrame"`
}

type Profile struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func Defaults() Settings {
	return Settings{Language: "en", Theme: "dark", Covers: true, Volume: 0.7}
}

type Store struct {
	Dir string
}

func Open(appName string) (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

func (s *Store) path(name string) string { return filepath.Join(s.Dir, name) }

func (s *Store) LoadSettings() (Settings, error) {
	st := Defaults()
	if err := s.readJSON("settings.json", &st); err != nil {
		return Defaults(), err
	}
	if st.Language != "ru" {
		st.Language = "en"
	}
	return st, nil
}

func (s *Store) SaveSettings(st Settings) error {
	return s.writeJSON("settings.json", st)
}

func (s *Store) readJSON(name string, v any) error {
	data, err := os.ReadFile(s.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (s *Store) writeJSON(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, name+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path(name))
}
