package recording

import (
	"slices"
	"testing"
	"time"
)

func TestOutputArgs(t *testing.T) {
	now := time.Date(2026, 11, 6, 14, 5, 34, 0, time.UTC)

	tests := []struct {
		name    string
		cfg     Config
		want    []string
		wantErr bool
	}{
		{
			name: "continuous",
			cfg:  Config{Mode: Continuous, OutputDirectory: "videos", FrameRate: 30},
			want: []string{"-n", "videos/dashcam_20261106-140534.mp4"},
		},
		{
			name: "continuous ignores zero duration and frame rate",
			cfg:  Config{Mode: Continuous, OutputDirectory: "videos"},
			want: []string{"-n", "videos/dashcam_20261106-140534.mp4"},
		},
		{
			name: "segmented 10 seconds at 30 fps",
			cfg:  Config{Mode: Segmented, SegmentDuration: 10 * time.Second, OutputDirectory: "videos", FrameRate: 30},
			want: []string{
				"-n",
				"-force_key_frames", "expr:gte(t,n_forced*10)",
				"-f", "segment",
				"-segment_time", "10",
				"-segment_time_delta", "0.016666666666666666",
				"-reset_timestamps", "1",
				"-strftime", "1",
				"videos/dashcam_%Y%m%d-%H%M%S.mp4",
			},
		},
		{
			name: "segmented fractional duration",
			cfg:  Config{Mode: Segmented, SegmentDuration: 1500 * time.Millisecond, OutputDirectory: "videos", FrameRate: 25},
			want: []string{
				"-n",
				"-force_key_frames", "expr:gte(t,n_forced*1.5)",
				"-f", "segment",
				"-segment_time", "1.5",
				"-segment_time_delta", "0.02",
				"-reset_timestamps", "1",
				"-strftime", "1",
				"videos/dashcam_%Y%m%d-%H%M%S.mp4",
			},
		},
		{
			name:    "segmented zero duration",
			cfg:     Config{Mode: Segmented, OutputDirectory: "videos", FrameRate: 30},
			wantErr: true,
		},
		{
			name:    "segmented negative duration",
			cfg:     Config{Mode: Segmented, SegmentDuration: -5 * time.Second, OutputDirectory: "videos", FrameRate: 30},
			wantErr: true,
		},
		{
			name:    "segmented zero frame rate",
			cfg:     Config{Mode: Segmented, SegmentDuration: 10 * time.Second, OutputDirectory: "videos"},
			wantErr: true,
		},
		{
			name:    "unknown mode",
			cfg:     Config{Mode: Mode(7), SegmentDuration: 10 * time.Second, OutputDirectory: "videos", FrameRate: 30},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OutputArgs(tt.cfg, now)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected an error, got args %q", got)
				}
				if got != nil {
					t.Errorf("Args should be nil on error, got %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("Args mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
