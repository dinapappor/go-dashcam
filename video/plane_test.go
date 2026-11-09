package video

import (
	"slices"
	"testing"
)

func TestAppendPlane(t *testing.T) {
	tests := []struct {
		name   string
		dst    []byte
		src    []byte
		stride int
		width  int
		height int
		want   []byte
	}{
		{
			name:   "tight plane excludes trailing data and preserves destination",
			dst:    []byte{1, 2},
			src:    []byte{10, 20, 30, 40, 99, 99},
			stride: 2, width: 2, height: 2,
			want: []byte{1, 2, 10, 20, 30, 40},
		},
		{
			name: "skip row padding",
			src: []byte{
				10, 20, 30, 99,
				40, 50, 60, 99,
			},
			stride: 4, width: 3, height: 2,
			want: []byte{10, 20, 30, 40, 50, 60},
		},
		{
			name:   "tightly packed rows",
			src:    []byte{10, 20, 30, 40, 50, 60},
			stride: 3, width: 3, height: 2,
			want: []byte{10, 20, 30, 40, 50, 60},
		},
		{
			name:   "preserve existing destination",
			dst:    []byte{1, 2},
			src:    []byte{10, 20, 99, 30, 40, 99},
			stride: 3, width: 2, height: 2,
			want: []byte{1, 2, 10, 20, 30, 40},
		},
		{
			name:   "last row needs no trailing padding",
			src:    []byte{10, 20, 99, 30, 40},
			stride: 3, width: 2, height: 2,
			want: []byte{10, 20, 30, 40},
		},
		{
			name:   "single pixel rows",
			src:    []byte{10, 99, 99, 20, 99, 99, 30},
			stride: 3, width: 1, height: 3,
			want: []byte{10, 20, 30},
		},
		{
			name:   "copy only requested rows",
			src:    []byte{10, 20, 99, 30, 40, 99},
			stride: 3, width: 2, height: 1,
			want: []byte{10, 20},
		},
		{
			name:   "zero height preserves destination",
			dst:    []byte{1, 2},
			stride: 3, width: 3, height: 0,
			want: []byte{1, 2},
		},
		{
			name:   "empty plane",
			stride: 0, width: 0, height: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSource := slices.Clone(tt.src)
			got := appendPlane(tt.dst, tt.src, tt.stride, tt.width, tt.height)
			if !slices.Equal(got, tt.want) {
				t.Errorf("appendPlane() = %v, want %v", got, tt.want)
			}
			if !slices.Equal(tt.src, originalSource) {
				t.Errorf("source changed to %v, want %v", tt.src, originalSource)
			}
		})
	}
}
