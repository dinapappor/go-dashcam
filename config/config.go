package config

import (
	"fmt"
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	DashcamTitle  string `json:"dashcam_title"`
	VideoConfig VideoConfig `json:"video_config"`
}

type VideoConfig struct {
	VideoSource     string `json:"video_source"` // camera
	Width           uint32 `json:"width"`
	Height          uint32 `json:"height"`
	FrameRate      uint32 `json:"fps"`
	Segmented       bool   `json:"segmented"`
	SegmentDuration string `json:"segment_duration"`
}

func ParseConfig(path string) (Config, error) {
	configFile, err := os.Open(path)

	if err != nil {
		return Config{}, fmt.Errorf("Unable to poen config file: %w", err)
	}

	defer configFile.Close()

	config := Config{}

	err = json.NewDecoder(configFile).Decode(&config)

	if err != nil {
		return Config{}, fmt.Errorf("Could not parse config: %w", err)
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


	return config, nil
}

func (c *Config) GetSegmentDuration() (time.Duration, error) {

	if ! c.VideoConfig.Segmented {
		// In continous mode we don't use duration
		return 0, nil
	}

	duration, err := time.ParseDuration(c.VideoConfig.SegmentDuration)

	if err != nil {
		return 0, fmt.Errorf("Could not parse time duration: %w", err)
	}
	return duration, nil

}
