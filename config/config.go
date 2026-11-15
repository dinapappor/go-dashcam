package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DashcamTitle    string      `json:"dashcam_title"`
	OutputDirectory string      `json:"output_directory"`
	SpeedUnit       string      `json:"speed_unit"`
	VideoConfig     VideoConfig `json:"video_config"`
	GPSConfig       *GPSConfig  `json:"gps_config"`
}

type VideoConfig struct {
	VideoSource     string `json:"video_source"` // camera - ie /dev/video0
	Width           uint32 `json:"width"`
	Height          uint32 `json:"height"`
	FrameRate       uint32 `json:"fps"`
	Segmented       bool   `json:"segmented"`
	SegmentDuration string `json:"segment_duration"`
	PixelFormat     string `json:"pixel_format"`
	Encoder         string `json:"encoder"`        // For now we make this hardcoded in config.
	EncoderDevice   string `json:"encoder_device"` // as above - should be auto-detected
	Bitrate         int32  `json:"bitrate"`
}

type GPSConfig struct {
	Type         string `json:"type"`
	Device       string `json:"device"`
	Address      string `json:"address"`
	AddressValue int    `json:"-"`        // a bit of cheating here
	Protocol     string `json:"protocol"` // ie, PMTK, ubx etc
}

func ParseConfig(path string) (Config, error) {
	configFile, err := os.Open(path)

	if err != nil {
		return Config{}, fmt.Errorf("Unable to open config file: %w", err)
	}

	defer configFile.Close()

	config := Config{}

	err = json.NewDecoder(configFile).Decode(&config)

	if err != nil {
		return Config{}, fmt.Errorf("Could not parse config: %w", err)
	}

	switch config.SpeedUnit {
	case "mph", "km/h":
		//do nothing
	case "kmh", "":
		config.SpeedUnit = "km/h"
	default:
		return Config{}, fmt.Errorf("Unknown speedunit %q", config.SpeedUnit)
	}

	if config.OutputDirectory == "" {
		config.OutputDirectory = "videos"
	}

	if config.GPSConfig != nil {
		config.GPSConfig.Type = strings.ToLower(config.GPSConfig.Type)
		switch config.GPSConfig.Type {
		case "i2c":
			// pa1010d or other i2c gps's
			tempAddr, err := strconv.ParseUint(config.GPSConfig.Address, 0, 8)

			if err != nil {
				return Config{}, fmt.Errorf("Could not parse gps address: %w", err)
			}

			config.GPSConfig.AddressValue = int(tempAddr)
		case "serial":
			// ublox etc
		default:
			return Config{}, fmt.Errorf("Unknown GPS type %s", config.GPSConfig.Type)
		}

		if config.GPSConfig.Device == "" {
			return Config{}, fmt.Errorf("GPS device must not be empty")
		}

		config.GPSConfig.Protocol = strings.ToLower(config.GPSConfig.Protocol)
		switch config.GPSConfig.Protocol {
		case "", "pmtk":
			//Cool

		default:
			return Config{}, fmt.Errorf("Unknown protocol %q", config.GPSConfig.Protocol)
		}
	}

	if config.VideoConfig.Width == 0 {
		return Config{}, fmt.Errorf("Config.VideoConfig.Width cannot be zero")
	}

	if config.VideoConfig.Height == 0 {
		return Config{}, fmt.Errorf("Config.VideoConfig.Height cannot be zero")
	}

	if config.VideoConfig.FrameRate == 0 {
		return Config{}, fmt.Errorf("Config.VideoConfig.FrameRate cannot be zero")
	}

	if config.VideoConfig.Segmented {
		segmentDuration, err := config.GetSegmentDuration()

		if err != nil {
			return Config{}, err
		}

		if segmentDuration <= 0 {
			return Config{}, fmt.Errorf("Config.VideoConfig.SegmentDuration cannot be zero or negative")
		}
	}

	switch config.VideoConfig.PixelFormat {
	case "MJPG", "YU12":
		// Automatically accept these formats
	case "":
		// Set to mjpeg
		config.VideoConfig.PixelFormat = "MJPG"
	default:
		return Config{}, fmt.Errorf("Unrecognised PixelFormat")
	}

	switch config.VideoConfig.Encoder {
	case "v4l2m2m", "software":
		// Don't fail on this. This is for the pi
	case "":
		config.VideoConfig.Encoder = "software"
	default:
		return Config{}, fmt.Errorf("Unrecognised encoder")
	}

	if config.VideoConfig.Encoder == "v4l2m2m" {
		if config.VideoConfig.EncoderDevice == "" {
			return Config{}, fmt.Errorf("EncoderDevice must be set when using v4l2m2m")
		}
		if config.VideoConfig.Bitrate < 0 {
			return Config{}, fmt.Errorf("Bitrate cannot be negative using v4l2m2m")
		}
		if config.VideoConfig.Bitrate == 0 {
			if config.VideoConfig.Height >= 1080 {
				config.VideoConfig.Bitrate = 10_000_000
			} else {
				config.VideoConfig.Bitrate = 5_000_000
			}
		}
	}

	return config, nil
}

func (c *Config) GetSegmentDuration() (time.Duration, error) {

	if !c.VideoConfig.Segmented {
		// In continous mode we don't use duration
		return 0, nil
	}

	duration, err := time.ParseDuration(c.VideoConfig.SegmentDuration)

	if err != nil {
		return 0, fmt.Errorf("Could not parse time duration: %w", err)
	}
	return duration, nil

}
