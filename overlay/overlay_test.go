package overlay

import (
	"image"
	"testing"
)

func TestRenderTitleDimensions(t *testing.T) {
	// Face7x13 uses seven pixels per character. Padding adds 10 by 5.
	tests := []struct {
		name  string
		title string
		width int
	}{
		{"empty title", "", 10},
		{"one character", "A", 17},
		{"multiple characters", "Dashcam", 59},
		{"includes space", "Go Cam", 52},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels, alpha := RenderTitle(Config{Title: tt.title})
			if pixels == nil || alpha == nil {
				t.Fatal("RenderTitle returned a nil image")
			}
			want := image.Rect(0, 0, tt.width, 18)
			if pixels.Bounds() != want {
				t.Errorf("brightness bounds = %v, want %v", pixels.Bounds(), want)
			}
			if alpha.Bounds() != want {
				t.Errorf("alpha bounds = %v, want %v", alpha.Bounds(), want)
			}
		})
	}
}

func TestRenderTitleBackgroundPadding(t *testing.T) {
	pixels, alpha := RenderTitle(Config{Title: "Andreas Dashcam"})
	bounds := pixels.Bounds()
	// Check the whole outer border, including all four corners.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x != bounds.Min.X && x != bounds.Max.X-1 && y != bounds.Min.Y && y != bounds.Max.Y-1 {
				continue
			}
			if got := pixels.GrayAt(x, y).Y; got != 0 {
				t.Errorf("brightness at (%d, %d) = %d, want black (0)", x, y, got)
			}
			if got := alpha.AlphaAt(x, y).A; got != 128 {
				t.Errorf("alpha at (%d, %d) = %d, want 128", x, y, got)
			}
		}
	}
}

func TestRenderTitleWhiteText(t *testing.T) {
	pixels, alpha := RenderTitle(Config{Title: "Andreas Dashcam"})
	whitePixels := 0
	for y := 0; y < pixels.Bounds().Dy(); y++ {
		for x := 0; x < pixels.Bounds().Dx(); x++ {
			brightness := pixels.GrayAt(x, y).Y
			opacity := alpha.AlphaAt(x, y).A
			switch brightness {
			case 255:
				whitePixels++
				if opacity != 255 {
					t.Errorf("white text at (%d, %d) has alpha %d, want 255", x, y, opacity)
				}
			case 0:
				if opacity != 128 {
					t.Errorf("black background at (%d, %d) has alpha %d, want 128", x, y, opacity)
				}
			default:
				t.Errorf("brightness at (%d, %d) = %d, want 0 or 255 for the bitmap font", x, y, brightness)
			}
		}
	}
	if whitePixels == 0 {
		t.Error("title contains no white text pixels")
	}
}

func TestRenderTitleEmptyText(t *testing.T) {
	pixels, alpha := RenderTitle(Config{Title: ""})
	for y := 0; y < pixels.Bounds().Dy(); y++ {
		for x := 0; x < pixels.Bounds().Dx(); x++ {
			if got := pixels.GrayAt(x, y).Y; got != 0 {
				t.Errorf("brightness at (%d, %d) = %d, want 0", x, y, got)
			}
			if got := alpha.AlphaAt(x, y).A; got != 128 {
				t.Errorf("alpha at (%d, %d) = %d, want 128", x, y, got)
			}
		}
	}
}
