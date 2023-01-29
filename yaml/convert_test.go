package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	home, err := os.UserHomeDir()
	assert.NoError(t, err)

	ankoMap := map[string]interface{}{
		"general.confirm_bulk_add":   true,
		"general.confirm_on_exit":    true,
		"general.queue_loop":         true,
		"general.load_prev_queue":    true,
		"general.popup_timeout":      "1s",
		"general.sort_by_mtime":      true,
		"general.music_dir":          filepath.Join(home, "Music"),
		"general.history_path":       filepath.Join(home, ".local", "share", "gomu", "urls"),
		"general.use_emoji":          true,
		"general.volume":             100,
		"general.invidious_instance": "https://yewtu.be",
		"general.lang_lyric":         "en",
		"general.rename_bytag":       true,
		"emoji.playlist":             "📁",
		"emoji.file":                 "🎵",
		"emoji.loop":                 "🔁",
		"emoji.noloop":               "🔂",
		"color.accent":               "red",
		"color.background":           "black",
		"color.foreground":           "white",
		"color.popup":                "blue",
		"color.playlist_directory":   "yellow",
		"color.playlist_highlight":   "green",
		"color.queue_highlight":      "cyan",
		"color.now_playing.title":    "magenta",
		"color.subtitle":             "white",
	}
	yamlMap, err := Convert(ankoMap)
	assert.NoError(t, err)
	assert.Equal(t, yamlMap, map[string]interface{}{
		"general": map[string]interface{}{
			"confirm_bulk_add":   true,
			"confirm_on_exit":    true,
			"queue_loop":         true,
			"load_prev_queue":    true,
			"popup_timeout":      "1s",
			"sort_by_mtime":      true,
			"music_dir":          filepath.Join(home, "Music"),
			"history_path":       filepath.Join(home, ".local", "share", "gomu", "urls"),
			"use_emoji":          true,
			"volume":             100,
			"invidious_instance": "https://yewtu.be",
			"lang_lyric":         "en",
			"rename_bytag":       true,
		},
		"emoji": map[string]interface{}{
			"playlist": "📁",
			"file":     "🎵",
			"loop":     "🔁",
			"noloop":   "🔂",
		},
		"color": map[string]interface{}{
			"accent":             "red",
			"background":         "black",
			"foreground":         "white",
			"popup":              "blue",
			"playlist_directory": "yellow",
			"playlist_highlight": "green",
			"queue_highlight":    "cyan",
			"now_playing": map[string]interface{}{
				"title": "magenta",
			},
			"subtitle": "white",
		},
	})
}
