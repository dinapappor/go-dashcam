package video

import (
	"image"
	"testing"
)

func TestValidateSubsampleRatio(t *testing.T) {
	tests := []struct {
		name     string
		ratio    image.YCbCrSubsampleRatio
		expected PixelFormat
		wantErr  bool
	}{
		{"422 matches YUV422", image.YCbCrSubsampleRatio422, YUV422, false},
		{"420 matches YUV420", image.YCbCrSubsampleRatio420, YUV420, false},
		{"420 mismatches YUV422", image.YCbCrSubsampleRatio420, YUV422, true},
		{"422 mismatches YUV420", image.YCbCrSubsampleRatio422, YUV420, true},
		{"444 mismatches YUV422", image.YCbCrSubsampleRatio444, YUV422, true},
		{"444 mismatches YUV420", image.YCbCrSubsampleRatio444, YUV420, true},
		{"440 mismatches YUV422", image.YCbCrSubsampleRatio440, YUV422, true},
		{"440 mismatches YUV420", image.YCbCrSubsampleRatio440, YUV420, true},
		{"411 mismatches YUV422", image.YCbCrSubsampleRatio411, YUV422, true},
		{"411 mismatches YUV420", image.YCbCrSubsampleRatio411, YUV420, true},
		{"410 mismatches YUV422", image.YCbCrSubsampleRatio410, YUV422, true},
		{"410 mismatches YUV420", image.YCbCrSubsampleRatio410, YUV420, true},
		{"unknown ratio with YUV422", image.YCbCrSubsampleRatio(99), YUV422, true},
		{"unknown ratio with YUV420", image.YCbCrSubsampleRatio(99), YUV420, true},
		{"unknown format with 422", image.YCbCrSubsampleRatio422, PixelFormat(99), true},
		{"unknown format with 420", image.YCbCrSubsampleRatio420, PixelFormat(99), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only the ratio matters here; no camera or pixel data is needed.
			frame := &image.YCbCr{SubsampleRatio: tt.ratio}
			err := ValidateSubsampleRatio(frame, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubsampleRatio(%v, %v) error = %v, wantErr %v",
					tt.ratio, tt.expected, err, tt.wantErr)
			}
		})
	}
}
