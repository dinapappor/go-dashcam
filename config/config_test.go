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

func TestParseConfigOutputDirectory(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{name: "Output directory defaults to videos", want: "videos"},
		{name: "Set outputdirectory is kept", field: `,"output_directory":"test123"`, want: "test123"},
		{name: "Empty outputdirectory gives videos", field: `,"output_directory":""`, want: "videos"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{"video_config":{"width":1280,"height":720,"fps":30}` + tt.field + `}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if cfg.OutputDirectory != tt.want {
				t.Errorf("OutputDirectory = %q, want %q", cfg.OutputDirectory, tt.want)
			}
		})
	}
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

func TestParseConfigEncoder(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		want        string
		wantDevice  string
		wantBitrate int32
		wantErr     bool
	}{
		{name: "missing defaults to software", want: "software"},
		{name: "empty defaults to software", field: `,"encoder":""`, want: "software"},
		{name: "explicit software", field: `,"encoder":"software"`, want: "software"},
		{name: "software needs no device or bitrate", field: `,"encoder":"software","bitrate":0`, want: "software"},
		{name: "v4l2m2m with device and bitrate", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":4000000`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 4000000},
		{name: "unknown encoder", field: `,"encoder":"nvenc"`, wantErr: true},
		{name: "v4l2m2m without device", field: `,"encoder":"v4l2m2m","bitrate":4000000`, wantErr: true},
		{name: "v4l2m2m with empty device", field: `,"encoder":"v4l2m2m","encoder_device":"","bitrate":4000000`, wantErr: true},
		{name: "v4l2m2m without bitrate defaults to 5M at 720p", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11"`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 5_000_000},
		{name: "v4l2m2m without bitrate defaults to 10M at 1080p", field: `,"height":1080,"encoder":"v4l2m2m","encoder_device":"/dev/video11"`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 10_000_000},
		{name: "v4l2m2m without bitrate defaults to 10M above 1080p", field: `,"height":1440,"encoder":"v4l2m2m","encoder_device":"/dev/video11"`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 10_000_000},
		{name: "v4l2m2m explicit bitrate is kept at 720p", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":8000000`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 8_000_000},
		{name: "v4l2m2m explicit bitrate is kept at 1080p", field: `,"height":1080,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":4000000`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 4_000_000},
		{name: "v4l2m2m with zero bitrate defaults to 5M at 720p", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":0`, want: "v4l2m2m", wantDevice: "/dev/video11", wantBitrate: 5_000_000},
		{name: "v4l2m2m with negative bitrate", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":-1`, wantErr: true},
		{name: "bitrate too large for int32", field: `,"encoder":"v4l2m2m","encoder_device":"/dev/video11","bitrate":3000000000`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{"video_config":{"width":1280,"height":720,"fps":30` + tt.field + `}}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got config %+v", cfg.VideoConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if cfg.VideoConfig.Encoder != tt.want {
				t.Errorf("Encoder = %q, want %q", cfg.VideoConfig.Encoder, tt.want)
			}
			if cfg.VideoConfig.EncoderDevice != tt.wantDevice {
				t.Errorf("EncoderDevice = %q, want %q", cfg.VideoConfig.EncoderDevice, tt.wantDevice)
			}
			if cfg.VideoConfig.Bitrate != tt.wantBitrate {
				t.Errorf("Bitrate = %d, want %d", cfg.VideoConfig.Bitrate, tt.wantBitrate)
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

func TestParseConfigSpeedUnit(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		want    string
		wantErr bool
	}{
		{name: "missing defaults to km/h", want: "km/h"},
		{name: "empty defaults to km/h", field: `"speed_unit":"",`, want: "km/h"},
		{name: "kmh becomes km/h", field: `"speed_unit":"kmh",`, want: "km/h"},
		{name: "explicit km/h", field: `"speed_unit":"km/h",`, want: "km/h"},
		{name: "explicit mph", field: `"speed_unit":"mph",`, want: "mph"},
		{name: "unknown unit", field: `"speed_unit":"knots",`, wantErr: true},
		{name: "wrong case", field: `"speed_unit":"MPH",`, wantErr: true},
		{name: "surrounding space", field: `"speed_unit":" kmh",`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{` + tt.field + `"video_config":{"width":1280,"height":720,"fps":30}}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unknown speed unit error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if cfg.SpeedUnit != tt.want {
				t.Errorf("SpeedUnit = %q, want %q", cfg.SpeedUnit, tt.want)
			}
		})
	}
}

func TestParseConfigGPS(t *testing.T) {
	tests := []struct {
		name         string
		field        string
		wantNil      bool
		wantDevice   string
		wantAddress  int
		wantProtocol string
		wantErr      bool
	}{
		{name: "missing block means no GPS", wantNil: true},
		{name: "missing protocol means plain NMEA", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10"},`, wantDevice: "/dev/i2c-1", wantAddress: 16, wantProtocol: ""},
		{name: "empty protocol means plain NMEA", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10","protocol":""},`, wantDevice: "/dev/i2c-1", wantAddress: 16, wantProtocol: ""},
		{name: "pmtk protocol", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10","protocol":"pmtk"},`, wantDevice: "/dev/i2c-1", wantAddress: 16, wantProtocol: "pmtk"},
		{name: "uppercase PMTK is lowered", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10","protocol":"PMTK"},`, wantDevice: "/dev/i2c-1", wantAddress: 16, wantProtocol: "pmtk"},
		{name: "ubx protocol", field: `"gps_config":{"type":"serial","device":"/dev/ttyACM0","protocol":"ubx"},`, wantDevice: "/dev/ttyACM0", wantProtocol: "ubx"},
		{name: "uppercase UBX is lowered", field: `"gps_config":{"type":"serial","device":"/dev/ttyACM0","protocol":"UBX"},`, wantDevice: "/dev/ttyACM0", wantProtocol: "ubx"},
		{name: "unknown protocol", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10","protocol":"sirf"},`, wantErr: true},
		{name: "hex address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x10"},`, wantDevice: "/dev/i2c-1", wantAddress: 16},
		{name: "decimal address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"16"},`, wantDevice: "/dev/i2c-1", wantAddress: 16},
		{name: "uppercase hex address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0X1F"},`, wantDevice: "/dev/i2c-1", wantAddress: 31},
		{name: "largest address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0xff"},`, wantDevice: "/dev/i2c-1", wantAddress: 255},
		{name: "unknown type", field: `"gps_config":{"type":"spi","device":"/dev/spidev0.0","address":"0x10"},`, wantErr: true},
		{name: "serial without address", field: `"gps_config":{"type":"serial","device":"/dev/ttyACM0"},`, wantDevice: "/dev/ttyACM0", wantAddress: 0},
		{name: "serial ignores address", field: `"gps_config":{"type":"serial","device":"/dev/ttyACM0","address":"zz"},`, wantDevice: "/dev/ttyACM0", wantAddress: 0},
		{name: "uppercase serial type", field: `"gps_config":{"type":"SERIAL","device":"/dev/ttyACM0"},`, wantDevice: "/dev/ttyACM0", wantAddress: 0},
		{name: "uppercase i2c type", field: `"gps_config":{"type":"I2C","device":"/dev/i2c-1","address":"0x10"},`, wantDevice: "/dev/i2c-1", wantAddress: 16},
		{name: "serial with pmtk protocol", field: `"gps_config":{"type":"serial","device":"/dev/serial0","protocol":"pmtk"},`, wantDevice: "/dev/serial0", wantProtocol: "pmtk"},
		{name: "serial empty device", field: `"gps_config":{"type":"serial"},`, wantErr: true},
		{name: "empty type", field: `"gps_config":{"device":"/dev/i2c-1","address":"0x10"},`, wantErr: true},
		{name: "empty device", field: `"gps_config":{"type":"i2c","address":"0x10"},`, wantErr: true},
		{name: "empty address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1"},`, wantErr: true},
		{name: "garbage address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"zz"},`, wantErr: true},
		{name: "address too large", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"0x100"},`, wantErr: true},
		{name: "negative address", field: `"gps_config":{"type":"i2c","device":"/dev/i2c-1","address":"-1"},`, wantErr: true},
		{name: "empty block", field: `"gps_config":{},`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{` + tt.field + `"video_config":{"width":1280,"height":720,"fps":30}}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected GPS config error, got %+v", cfg.GPSConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if tt.wantNil {
				if cfg.GPSConfig != nil {
					t.Errorf("GPSConfig = %+v, want nil", cfg.GPSConfig)
				}
				return
			}
			if cfg.GPSConfig == nil {
				t.Fatal("GPSConfig is nil")
			}
			if cfg.GPSConfig.Device != tt.wantDevice {
				t.Errorf("Device = %q, want %q", cfg.GPSConfig.Device, tt.wantDevice)
			}
			if cfg.GPSConfig.AddressValue != tt.wantAddress {
				t.Errorf("AddressValue = %d, want %d", cfg.GPSConfig.AddressValue, tt.wantAddress)
			}
			if cfg.GPSConfig.Protocol != tt.wantProtocol {
				t.Errorf("Protocol = %q, want %q", cfg.GPSConfig.Protocol, tt.wantProtocol)
			}
		})
	}
}

func TestParseConfigGPSAiding(t *testing.T) {
	tests := []struct {
		name    string
		aiding  string
		want    bool
		wantErr bool
	}{
		{name: "missing defaults to on", aiding: ``, want: true},
		{name: "explicit true", aiding: `,"aiding":true`, want: true},
		{name: "explicit false", aiding: `,"aiding":false`, want: false},
		{name: "null defaults to on", aiding: `,"aiding":null`, want: true},
		{name: "string is rejected", aiding: `,"aiding":"yes"`, wantErr: true},
		{name: "number is rejected", aiding: `,"aiding":1`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := `{"gps_config":{"type":"serial","device":"/dev/ttyACM0","protocol":"ubx"` + tt.aiding + `},"video_config":{"width":1280,"height":720,"fps":30}}`
			cfg, err := ParseConfig(writeConfigFile(t, contents))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected aiding error, got %+v", cfg.GPSConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if cfg.GPSConfig == nil {
				t.Fatal("GPSConfig is nil")
			}
			if cfg.GPSConfig.Aiding == nil {
				t.Fatal("Aiding is nil, want it filled in by ParseConfig")
			}
			if got := *cfg.GPSConfig.Aiding; got != tt.want {
				t.Errorf("Aiding = %t, want %t", got, tt.want)
			}
		})
	}
}
