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
	FramesPerSecond uint32 `json:"fps"`
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

	return config, nil
}

func (c *Config) GetSegmentDuration() (time.Duration, error) {

	duration, err := time.ParseDuration(c.VideoConfig.SegmentDuration)

	if err != nil {
		return 0, fmt.Errorf("Could not parse time duration: %w", err)
	}
	return duration, nil

}
