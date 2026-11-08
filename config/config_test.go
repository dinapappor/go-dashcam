package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseConfigPixelFormat(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		want    string
		wantErr bool
	}{
		{name: "missing defaults to MJPG", want: "MJPG"},
		{name: "empty defaults to MJPG", field: `,"pixel_format":""`, want: "MJPG"},
		{name: "explicit MJPG", field: `,"pixel_format":"MJPG"`, want: "MJPG"},
		{name: "explicit YU12", field: `,"pixel_format":"YU12"`, want: "YU12"},
		{name: "unsupported format", field: `,"pixel_format":"YUYV"`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{"video_config":{"width":1280,"height":720,"fps":30` + tt.field + `}}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unsupported pixel format error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if cfg.VideoConfig.PixelFormat != tt.want {
				t.Errorf("PixelFormat = %q, want %q", cfg.VideoConfig.PixelFormat, tt.want)
			}
		})
	}
}

func TestParseConfigValid(t *testing.T) {
	path := writeConfigFile(t, `{"dashcam_title":"Andreas Dashcam","video_config":{"video_source":"/dev/video4","width":1280,"height":720,"fps":30,"segmented":true,"segment_duration":"1.5s"}}`)
	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DashcamTitle != "Andreas Dashcam" || cfg.VideoConfig.VideoSource != "/dev/video4" || cfg.VideoConfig.Width != 1280 || cfg.VideoConfig.Height != 720 || !cfg.VideoConfig.Segmented || cfg.VideoConfig.SegmentDuration != "1.5s" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	// The on-disk key stays "fps", regardless of the Go field's name.
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Video struct {
			FPS uint32 `json:"fps"`
		} `json:"video_config"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Video.FPS != 30 {
		t.Errorf("saved fps = %d, want 30", saved.Video.FPS)
	}
}

func TestParseConfigMissingFile(t *testing.T) {
	cfg, err := ParseConfig(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want os.ErrNotExist", err)
	}
	if cfg != (Config{}) {
		t.Errorf("config = %+v, want zero value", cfg)
	}
}

func TestParseConfigInvalidJSON(t *testing.T) {
	tests := []struct{ name, contents string }{
		{"empty file", ""},
		{"malformed JSON", `{"video_config":`},
		{"wrong width type", `{"video_config":{"width":"wide"}}`},
		{"negative unsigned width", `{"video_config":{"width":-1}}`},
		{"width overflow", `{"video_config":{"width":4294967296}}`},
		{"invalid recording mode type", `{"video_config":{"segmented":"yes"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(writeConfigFile(t, tt.contents))
			if err == nil {
				t.Fatal("expected an error")
			}
			if cfg != (Config{}) {
				t.Errorf("config = %+v, want zero value", cfg)
			}
		})
	}
}

func TestParseConfigValidation(t *testing.T) {
	tests := []struct{ name, contents string }{
		{"zero width", `{"video_config":{"width":0,"height":720,"fps":30,"segmented":true,"segment_duration":"10s"}}`},
		{"zero height", `{"video_config":{"width":1280,"height":0,"fps":30,"segmented":true,"segment_duration":"10s"}}`},
		{"zero frame rate", `{"video_config":{"width":1280,"height":720,"fps":0,"segmented":true,"segment_duration":"10s"}}`},
		{"invalid duration", `{"video_config":{"width":1280,"height":720,"fps":30,"segmented":true,"segment_duration":"soon"}}`},
		{"zero duration", `{"video_config":{"width":1280,"height":720,"fps":30,"segmented":true,"segment_duration":"0s"}}`},
		{"negative duration", `{"video_config":{"width":1280,"height":720,"fps":30,"segmented":true,"segment_duration":"-1s"}}`},
		{"missing duration", `{"video_config":{"width":1280,"height":720,"fps":30,"segmented":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseConfig(writeConfigFile(t, tt.contents)); err == nil {
				t.Error("expected invalid settings to be rejected")
			}
		})
	}
}

func TestGetSegmentDuration(t *testing.T) {
	tests := []struct {
		name, value string
		want        time.Duration
		wantErr     bool
	}{
		{"seconds", "10s", 10 * time.Second, false},
		{"fractional seconds", "1.5s", 1500 * time.Millisecond, false},
		{"minutes", "2m", 2 * time.Minute, false},
		{"invalid", "soon", 0, true},
		{"missing unit", "10", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{VideoConfig: VideoConfig{Segmented: true, SegmentDuration: tt.value}}
			got, err := cfg.GetSegmentDuration()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("duration = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContinuousConfigWithoutDuration(t *testing.T) {
	cfg, err := ParseConfig(writeConfigFile(t, `{"video_config":{"width":1280,"height":720,"fps":30,"segmented":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	// Startup calls this even in continuous mode, where no duration is needed.
	got, err := cfg.GetSegmentDuration()
	if err != nil {
		t.Fatalf("continuous mode should not require a duration: %v", err)
	}
	if got != 0 {
		t.Errorf("duration = %v, want 0", got)
	}
}
