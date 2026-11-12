package overlay

import (
	"image"
	"slices"
	"testing"
)

func TestBlendPixelTransparent(t *testing.T) {
	got := blendPixel(100, 255, 0)

	if got != 100 {
		t.Errorf("blendPixel() = %d, want 100", got)
	}
}

func TestBlendPixelFullyOpaque(t *testing.T) {
	got := blendPixel(100, 255, 255)

	if got != 255 {
		t.Errorf("blendPixel() = %d, want 255", got)
	}
}

func TestBlendPixelPartialTransparency(t *testing.T) {
	got := blendPixel(100, 255, 128)

	// (255×128 + 100×127) / 255 = 177
	if got != 177 {
		t.Errorf("blendPixel() = %d, want 177", got)
	}
}

func TestBlendPixelMaximumBrightness(t *testing.T) {
	got := blendPixel(255, 255, 128)

	if got != 255 {
		t.Errorf("blendPixel() = %d, want 255", got)
	}
}

func TestBlendOverlayPlacement(t *testing.T) {
	frame := []byte{100, 100, 100, 100, 100, 100}

	title := image.NewGray(image.Rect(0, 0, 1, 1))
	title.Pix[0] = 255

	alpha := image.NewAlpha(image.Rect(0, 0, 1, 1))
	alpha.Pix[0] = 255

	BlendOverlay(frame, 3, 3, 2, title, alpha, 1, 0)

	want := []byte{100, 255, 100, 100, 100, 100}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayRightEdge(t *testing.T) {
	frame := []byte{100, 100, 100, 100, 100, 100}

	title := image.NewGray(image.Rect(0, 0, 2, 1))
	title.Pix[0] = 255
	title.Pix[1] = 255

	alpha := image.NewAlpha(image.Rect(0, 0, 2, 1))
	alpha.Pix[0] = 255
	alpha.Pix[1] = 255

	BlendOverlay(frame, 3, 3, 2, title, alpha, 2, 0)

	want := []byte{100, 100, 255, 100, 100, 100}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayBottomEdge(t *testing.T) {
	frame := []byte{100, 100, 100, 100, 100, 100}

	title := image.NewGray(image.Rect(0, 0, 1, 2))
	title.Pix[0] = 255
	title.Pix[1] = 255

	alpha := image.NewAlpha(image.Rect(0, 0, 1, 2))
	alpha.Pix[0] = 255
	alpha.Pix[1] = 255

	BlendOverlay(frame, 3, 3, 2, title, alpha, 1, 1)

	want := []byte{100, 100, 100, 100, 255, 100}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendPixelRounding(t *testing.T) {
	// Half of one brightness step is truncated to zero, not rounded up.
	if got := blendPixel(0, 1, 128); got != 0 {
		t.Errorf("blendPixel() = %d, want 0", got)
	}
}

func TestBlendPixelBlackBackground(t *testing.T) {
	if got := blendPixel(100, 0, 128); got != 49 {
		t.Errorf("blendPixel() = %d, want 49", got)
	}
}

func TestBlendOverlayFrameStride(t *testing.T) {
	// Each row has three visible pixels and one padding byte.
	frame := []byte{
		100, 100, 100, 99,
		100, 100, 100, 99,
	}
	title := &image.Gray{Pix: []byte{255, 255}, Stride: 1, Rect: image.Rect(0, 0, 1, 2)}
	alpha := &image.Alpha{Pix: []byte{255, 255}, Stride: 1, Rect: image.Rect(0, 0, 1, 2)}

	BlendOverlay(frame, 4, 3, 2, title, alpha, 1, 0)

	want := []byte{
		100, 255, 100, 99,
		100, 255, 100, 99,
	}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayMixedTransparency(t *testing.T) {
	frame := []byte{100, 100, 100, 100}
	title := &image.Gray{Pix: []byte{255, 255, 255, 0}, Stride: 4, Rect: image.Rect(0, 0, 4, 1)}
	alpha := &image.Alpha{Pix: []byte{0, 255, 128, 128}, Stride: 4, Rect: image.Rect(0, 0, 4, 1)}

	BlendOverlay(frame, 4, 4, 1, title, alpha, 0, 0)

	want := []byte{100, 255, 177, 49}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayIndependentImageStrides(t *testing.T) {
	frame := []byte{100, 100, 100, 100}
	// Title rows are three bytes apart; alpha rows are four bytes apart.
	// Padding must not be read as brightness or opacity.
	title := &image.Gray{
		Pix:    []byte{200, 150, 99, 80, 60, 99},
		Stride: 3, Rect: image.Rect(0, 0, 2, 2),
	}
	alpha := &image.Alpha{
		Pix:    []byte{255, 0, 0, 0, 255, 128, 0, 0},
		Stride: 4, Rect: image.Rect(0, 0, 2, 2),
	}

	BlendOverlay(frame, 2, 2, 2, title, alpha, 0, 0)

	want := []byte{200, 100, 80, 79}
	if !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		offsetX int
		offsetY int
		want    []byte
	}{
		{"exact fit", 0, 0, []byte{10, 20, 30, 40}},
		{"bottom right corner", 1, 1, []byte{100, 100, 100, 10}},
		{"entirely right", 2, 0, []byte{100, 100, 100, 100}},
		{"entirely below", 0, 2, []byte{100, 100, 100, 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := []byte{100, 100, 100, 100}
			title := &image.Gray{Pix: []byte{10, 20, 30, 40}, Stride: 2, Rect: image.Rect(0, 0, 2, 2)}
			alpha := &image.Alpha{Pix: []byte{255, 255, 255, 255}, Stride: 2, Rect: image.Rect(0, 0, 2, 2)}

			BlendOverlay(frame, 2, 2, 2, title, alpha, tt.offsetX, tt.offsetY)

			if !slices.Equal(frame, tt.want) {
				t.Errorf("frame = %v, want %v", frame, tt.want)
			}
		})
	}
}

func TestBlendOverlayEmptyTitle(t *testing.T) {
	frame := []byte{100, 100}
	title := image.NewGray(image.Rect(0, 0, 0, 0))
	alpha := image.NewAlpha(title.Bounds())

	BlendOverlay(frame, 2, 2, 1, title, alpha, 0, 0)

	if want := []byte{100, 100}; !slices.Equal(frame, want) {
		t.Errorf("frame = %v, want %v", frame, want)
	}
}

func TestBlendOverlayEmptyFrame(t *testing.T) {
	frame := []byte{}
	title := &image.Gray{Pix: []byte{255}, Stride: 1, Rect: image.Rect(0, 0, 1, 1)}
	alpha := &image.Alpha{Pix: []byte{255}, Stride: 1, Rect: title.Bounds()}

	BlendOverlay(frame, 0, 0, 0, title, alpha, 0, 0)

	if len(frame) != 0 {
		t.Errorf("frame = %v, want empty", frame)
	}
}
