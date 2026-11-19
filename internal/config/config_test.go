package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	s := &Store{Dir: t.TempDir()}

	st, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if st != Defaults() || st.Language != "en" {
		t.Fatalf("первый запуск: %+v", st)
	}

	st.Language, st.Theme = "ru", "neon"
	st.Profile = Profile{Nickname: "zeroqual", Avatar: "avatar.png"}
	if err := s.SaveSettings(st); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadSettings()
	if err != nil || got != st {
		t.Fatalf("прочитали %+v, %v; ожидали %+v", got, err, st)
	}

	files, _ := filepath.Glob(filepath.Join(s.Dir, "*.tmp"))
	if len(files) != 0 {
		t.Fatalf("остались временные файлы: %v", files)
	}
}

func TestBrokenSettingsFallBackToDefaults(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(s.Dir, "settings.json"), []byte("{oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadSettings()
	if err == nil {
		t.Fatal("ожидали ошибку разбора")
	}
	if st != Defaults() {
		t.Fatalf("при ошибке ждём настройки по умолчанию, а получили %+v", st)
	}
}

func TestPlaylists(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	p, err := s.LoadPlaylists()
	if err != nil || len(p.Items) != 0 {
		t.Fatalf("%+v %v", p, err)
	}

	if _, err := p.Create("   "); err != ErrEmptyName {
		t.Fatalf("пустое имя: %v", err)
	}
	id, err := p.Create("Work")
	if err != nil {
		t.Fatal(err)
	}
	pl, _ := p.Get(id)
	pl.Add("/a.mp3")
	pl.Add("/b.mp3")
	pl.Add("/a.mp3")
	pl.Remove("/b.mp3")
	if err := p.Rename(id, " Focus "); err != nil {
		t.Fatal(err)
	}

	if err := s.SavePlaylists(p); err != nil {
		t.Fatal(err)
	}
	p2, err := s.LoadPlaylists()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p2.Get(id)
	if !ok || got.Name != "Focus" || len(got.Tracks) != 1 || got.Tracks[0] != "/a.mp3" {
		t.Fatalf("после сохранения: %+v", got)
	}

	p2.Delete(id)
	if _, ok := p2.Get(id); ok {
		t.Fatal("плейлист не удалился")
	}
	if err := p2.Rename(id, "x"); err != ErrNotFound {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}
}

func TestRemoveAndPrune(t *testing.T) {
	p := &Playlists{}
	a, _ := p.Create("A")
	b, _ := p.Create("B")
	pa, _ := p.Get(a)
	pb, _ := p.Get(b)
	pa.Add("/m/1.mp3")
	pa.Add("/m/2.mp3")
	pb.Add("/m/2.mp3")
	pb.Add("/other/3.mp3")

	if !p.RemoveTrack("/m/2.mp3") {
		t.Fatal("RemoveTrack должен сообщить об изменении")
	}
	if pa.Contains("/m/2.mp3") || pb.Contains("/m/2.mp3") {
		t.Fatal("трек остался в плейлисте")
	}
	if p.RemoveTrack("/m/2.mp3") {
		t.Fatal("повторное удаление ничего не меняет")
	}

	changed := p.Prune(func(path string) bool { return path == "/other/3.mp3" })
	if !changed || len(pb.Tracks) != 0 || len(pa.Tracks) != 1 {
		t.Fatalf("Prune: %v %v %v", changed, pa.Tracks, pb.Tracks)
	}
}

func TestUniqueName(t *testing.T) {
	p := &Playlists{}
	_, _ = p.Create("Chill")
	_, _ = p.Create("chill (2)")
	if got := p.UniqueName("Gym"); got != "Gym" {
		t.Fatal(got)
	}
	if got := p.UniqueName("Chill"); got != "Chill (3)" {
		t.Fatalf("got %q", got)
	}
}
