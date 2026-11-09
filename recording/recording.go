package recording

import (
	"fmt"
	"path/filepath"
	"time"
)

type Mode int

const (
	Continuous Mode = iota
	Segmented
)

type Config struct {
	Mode            Mode
	SegmentDuration time.Duration
	OutputDirectory string
	FrameRate       uint32
	StreamCopy      bool
}

func OutputArgs(cfg Config, now time.Time) ([]string, error) {

	// First argument is to not overwrite files
	outputArgs := []string{"-n"}

	switch cfg.Mode {
		case Continuous:
			outputFile := fmt.Sprintf("dashcam_%s.mp4", now.Format("20060102-150405"))
			outputArgs = append(outputArgs, filepath.Join(cfg.OutputDirectory, outputFile))
		case Segmented:
			if cfg.FrameRate <= 0 {
				return nil, fmt.Errorf("We need framerate to accurately calculate key frames and segment delta")
			}
			if cfg.SegmentDuration <= 0 {
				return nil, fmt.Errorf("We need a positive segment duration or else we divide by zero")
			}
			if ! cfg.StreamCopy {
				outputArgs = append(
					outputArgs,
					"-force_key_frames",
					fmt.Sprintf("expr:gte(t,n_forced*%g)", cfg.SegmentDuration.Seconds()),
				)
			}
			outputArgs = append(
				outputArgs,
				"-f", "segment",
				"-segment_time", fmt.Sprint(cfg.SegmentDuration.Seconds()),
				"-segment_time_delta", fmt.Sprint(1.0/(2*float64(cfg.FrameRate))),
				"-reset_timestamps", "1",
				"-strftime", "1",
				filepath.Join(cfg.OutputDirectory,"dashcam_%Y%m%d-%H%M%S.mp4"),
			)
		default:
			return nil, fmt.Errorf("unknown recording mode %d", cfg.Mode)
	}
	return outputArgs, nil
}
