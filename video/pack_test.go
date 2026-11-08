package video

import (
	"image"
	"slices"
	"testing"
)

func TestPackFrame(t *testing.T) {
	// Each fixture starts at (0, 0), as jpeg.Decode images do.
	// Distinct plane values make swapped or missing colour planes visible.
	tests := []struct {
		name  string
		frame image.YCbCr
		want  []byte
	}{
		{
			name: "422 plane order",
			frame: image.YCbCr{
				Y:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Cb: []byte{11, 12, 13, 14}, Cr: []byte{21, 22, 23, 24},
				YStride: 4, CStride: 2,
				Rect: image.Rect(0, 0, 4, 2), SubsampleRatio: image.YCbCrSubsampleRatio422,
			},
			want: []byte{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14, 21, 22, 23, 24},
		},
		{
			name: "420 plane order",
			frame: image.YCbCr{
				Y:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Cb: []byte{11, 12}, Cr: []byte{21, 22},
				YStride: 4, CStride: 2,
				Rect: image.Rect(0, 0, 4, 2), SubsampleRatio: image.YCbCrSubsampleRatio420,
			},
			want: []byte{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 21, 22},
		},
		{
			name: "422 odd width and padded planes",
			frame: image.YCbCr{
				Y:  []byte{1, 2, 3, 99, 4, 5, 6},
				Cb: []byte{11, 12, 99, 13, 14}, Cr: []byte{21, 22, 99, 23, 24},
				YStride: 4, CStride: 3,
				Rect: image.Rect(0, 0, 3, 2), SubsampleRatio: image.YCbCrSubsampleRatio422,
			},
			want: []byte{1, 2, 3, 4, 5, 6, 11, 12, 13, 14, 21, 22, 23, 24},
		},
		{
			name: "420 odd width and height with padded planes",
			frame: image.YCbCr{
				Y:  []byte{1, 2, 3, 99, 4, 5, 6, 99, 7, 8, 9},
				Cb: []byte{11, 12, 99, 13, 14}, Cr: []byte{21, 22, 99, 23, 24},
				YStride: 4, CStride: 3,
				Rect: image.Rect(0, 0, 3, 3), SubsampleRatio: image.YCbCrSubsampleRatio420,
			},
			want: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 21, 22, 23, 24},
		},
		{
			name: "single pixel 420",
			frame: image.YCbCr{
				Y: []byte{1}, Cb: []byte{11}, Cr: []byte{21},
				YStride: 1, CStride: 1,
				Rect: image.Rect(0, 0, 1, 1), SubsampleRatio: image.YCbCrSubsampleRatio420,
			},
			want: []byte{1, 11, 21},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalY := slices.Clone(tt.frame.Y)
			originalCb := slices.Clone(tt.frame.Cb)
			originalCr := slices.Clone(tt.frame.Cr)
			got, err := PackFrame(nil, &tt.frame)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("PackFrame() = %v, want %v", got, tt.want)
			}
			if !slices.Equal(tt.frame.Y, originalY) || !slices.Equal(tt.frame.Cb, originalCb) || !slices.Equal(tt.frame.Cr, originalCr) {
				t.Error("PackFrame changed the source planes")
			}
		})
	}
}

func TestPackFrameReusesDestination(t *testing.T) {
	// A previous, longer frame must not leave a prefix or trailing bytes.
	dst := make([]byte, 20, 32)
	for i := range dst {
		dst[i] = 99
	}
	frame := &image.YCbCr{
		Y: []byte{1, 2}, Cb: []byte{11}, Cr: []byte{21},
		YStride: 2, CStride: 1,
		Rect: image.Rect(0, 0, 2, 1), SubsampleRatio: image.YCbCrSubsampleRatio422,
	}
	got, err := PackFrame(dst, frame)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 11, 21}
	if !slices.Equal(got, want) {
		t.Fatalf("PackFrame() = %v, want %v", got, want)
	}
	if &got[0] != &dst[0] {
		t.Error("PackFrame allocated a new buffer despite sufficient capacity")
	}
}

func TestPackFrameUnsupportedRatio(t *testing.T) {
	for _, ratio := range []image.YCbCrSubsampleRatio{
		image.YCbCrSubsampleRatio444,
		image.YCbCrSubsampleRatio440,
		image.YCbCrSubsampleRatio411,
		image.YCbCrSubsampleRatio410,
		image.YCbCrSubsampleRatio(99),
	} {
		t.Run(ratio.String(), func(t *testing.T) {
			frame := &image.YCbCr{
				Y: []byte{1, 2, 3, 4}, Cb: []byte{11, 12, 13, 14}, Cr: []byte{21, 22, 23, 24},
				YStride: 2, CStride: 2,
				Rect: image.Rect(0, 0, 2, 2), SubsampleRatio: ratio,
			}
			got, err := PackFrame(nil, frame)
			if err == nil {
				t.Error("expected an error for an unsupported ratio")
			}
			if got != nil {
				t.Errorf("PackFrame() = %v, want nil on error", got)
			}
		})
	}
}
