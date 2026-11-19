package i18n

import "testing"

func TestTracks(t *testing.T) {
	ru, en := New(RU), New(EN)
	cases := map[int][2]string{
		0:   {"0 треков", "0 tracks"},
		1:   {"1 трек", "1 track"},
		3:   {"3 трека", "3 tracks"},
		5:   {"5 треков", "5 tracks"},
		11:  {"11 треков", "11 tracks"},
		21:  {"21 трек", "21 tracks"},
		112: {"112 треков", "112 tracks"},
	}
	for n, want := range cases {
		if got := ru.Tracks(n); got != want[0] {
			t.Errorf("ru %d: %q", n, got)
		}
		if got := en.Tracks(n); got != want[1] {
			t.Errorf("en %d: %q", n, got)
		}
	}
}

func TestFallbacks(t *testing.T) {
	if New("de").Lang() != EN {
		t.Error("неизвестный язык должен стать английским")
	}
	if New(RU).S("no.such.key") != "no.such.key" {
		t.Error("забытый ключ должен показываться как есть")
	}
}

func TestRussianComplete(t *testing.T) {
	for k := range dict[EN] {
		if _, ok := dict[RU][k]; !ok {
			t.Errorf("нет перевода: %s", k)
		}
	}
	for k := range dict[RU] {
		if _, ok := dict[EN][k]; !ok {
			t.Errorf("лишний ключ в RU: %s", k)
		}
	}
}
