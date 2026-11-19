package library

import "testing"

func TestYouTubeID(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ":       "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                      "dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=dQw4w9WgXcQ&t=1": "dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=short":             "",
		"https://example.com/watch?v=dQw4w9WgXcQ":           "",
		"":                             "",
		"https://youtu.be/dQw4w9WgX<Q": "",
	}
	for in, want := range cases {
		if got := YouTubeID(in); got != want {
			t.Errorf("YouTubeID(%q) = %q, ожидали %q", in, got, want)
		}
	}
}
