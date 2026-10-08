// Package config keeps the settings of AltPaint in ~/.altbins/.altpaint
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Settings are the application options and what is remembered between starts
type Settings struct {
	// Language of the interface as a tag like "ru"; "" - the system's
	Language string `json:",omitempty"`

	// Color theme: "light"; "" - the dark one
	Theme string `json:",omitempty"`

	// PixelGrid shows the borders of the pixels when zoomed in
	PixelGrid bool `json:",omitempty"`

	// JPEGQuality is used when saving JPEG files, 1..100
	JPEGQuality int `json:",omitempty"`

	// RecentFiles are the files opened or saved last, the newest first
	RecentFiles []string `json:",omitempty"`

	// Tools are the options of the tools as they were left
	Tools ToolSettings
}

// ToolSettings are the options of the tool bar and the colors
type ToolSettings struct {
	Tool      string `json:",omitempty"`
	Primary   string `json:",omitempty"` // #RRGGBBAA
	Secondary string `json:",omitempty"`
	Width     float64
	Antialias bool
	Blend     bool // alpha blending; off - the pixels are replaced
	Fill      int  // the shape fill mode
	Shape     int
	Gradient  int
	Tolerance float64 // 0..100
	FloodMode int     // 0 contiguous, 1 global
	SampleAll bool    // the fill and the magic wand look at the image, not the layer
	FontSize  float64
	Bold      bool
	Italic    bool
	Mono      bool
}

// DefaultJPEGQuality is the quality of the JPEG files until it is changed
const DefaultJPEGQuality = 95

// RecentLimit is how many recent files are remembered
const RecentLimit = 10

var (
	settingsMtx sync.Mutex
	settings    = defaultSettings()
)

func defaultSettings() Settings {
	return Settings{
		JPEGQuality: DefaultJPEGQuality,
		Tools: ToolSettings{
			Tool:      "Paintbrush",
			Primary:   "#000000FF",
			Secondary: "#FFFFFFFF",
			Width:     2,
			Antialias: true,
			Blend:     true,
			Tolerance: 50,
			FontSize:  24,
		},
	}
}

// ConfigDirectory returns ~/.altbins/.altpaint
func ConfigDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".altbins", ".altpaint")
}

func settingsPath() string {
	return filepath.Join(ConfigDirectory(), "settings.json")
}

// LoadSettings reads the settings file; missing values get the defaults
func LoadSettings() {
	s := defaultSettings()
	if bs, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(bs, &s)
	}
	if s.JPEGQuality < 1 || s.JPEGQuality > 100 {
		s.JPEGQuality = DefaultJPEGQuality
	}
	settingsMtx.Lock()
	settings = s
	settingsMtx.Unlock()
}

// GetSettings returns the current settings; safe to call from any goroutine
func GetSettings() Settings {
	settingsMtx.Lock()
	defer settingsMtx.Unlock()
	s := settings
	s.RecentFiles = slices.Clone(s.RecentFiles)
	return s
}

// SetSettings applies and saves the settings
func SetSettings(s Settings) error {
	settingsMtx.Lock()
	settings = s
	settingsMtx.Unlock()

	bs, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ConfigDirectory(), 0755); err != nil {
		return err
	}
	return writeFileAtomic(settingsPath(), bs)
}

// UpdateSettings changes the settings with f and saves them
func UpdateSettings(f func(s *Settings)) error {
	s := GetSettings()
	f(&s)
	return SetSettings(s)
}

// AddRecentFile puts the file first in the recent files and saves the settings
func AddRecentFile(path string) {
	UpdateSettings(func(s *Settings) {
		s.RecentFiles = slices.DeleteFunc(s.RecentFiles, func(p string) bool { return p == path })
		s.RecentFiles = append([]string{path}, s.RecentFiles...)
		if len(s.RecentFiles) > RecentLimit {
			s.RecentFiles = s.RecentFiles[:RecentLimit]
		}
	})
}

// writeFileAtomic writes to a temporary file and renames it over the target,
// so a crash leaves either the old file or the new one, never a torn one
func writeFileAtomic(fullPath string, bs []byte) error {
	tmpPath := fullPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, err = f.Write(bs)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, fullPath)
	}
	if err != nil {
		os.Remove(tmpPath)
	}
	return err
}
