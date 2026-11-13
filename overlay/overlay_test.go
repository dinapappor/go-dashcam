package overlay

import (
	"testing"
)

func TestRenderOverlayDimensions(t *testing.T) {
	titles := []string{"", "A", "Go Cam", "Dashcam"}
	var widths []int
	height := 0
	for _, title := range titles {
		pixels, alpha, err := RenderOverlay(Config{Text: title, FrameHeight: 720})
		if err != nil {
			t.Fatalf("Error creating title %q: %v", title, err)
		}
		if pixels == nil || alpha == nil {
			t.Fatalf("RenderOverlay(%q) returned a nil image", title)
		}
		if pixels.Bounds() != alpha.Bounds() {
			t.Errorf("%q: brightness bounds %v differ from alpha bounds %v", title, pixels.Bounds(), alpha.Bounds())
		}
		if height == 0 {
			height = pixels.Bounds().Dy()
		}
		if pixels.Bounds().Dy() != height {
			t.Errorf("%q: height = %d, want %d like the other titles", title, pixels.Bounds().Dy(), height)
		}
		widths = append(widths, pixels.Bounds().Dx())
	}
	if widths[0] != 10 {
		t.Errorf("empty title width = %d, want 10 (padding only)", widths[0])
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] <= widths[i-1] {
			t.Errorf("%q width %d is not wider than %q width %d", titles[i], widths[i], titles[i-1], widths[i-1])
		}
	}
	if height <= 5 {
		t.Errorf("height = %d, want more than the 5 pixel padding", height)
	}
}

func TestRenderOverlayBackgroundPadding(t *testing.T) {
	pixels, alpha, err := RenderOverlay(Config{Text: "Andreas Dashcam", FrameHeight: 720})

	if err != nil {
		t.Fatalf("Error creating title: %v", err)
	}

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

func TestRenderOverlayWhiteText(t *testing.T) {
	pixels, alpha, err := RenderOverlay(Config{Text: "Andreas Dashcam", FrameHeight: 720})

	if err != nil {
		t.Fatalf("Error creating title: %v", err)
	}

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
				if opacity < 128 {
					t.Errorf("anti-aliased edge at (%d, %d) has alpha %d, want 128 to 255", x, y, opacity)
				}
			}
		}
	}
	if whitePixels == 0 {
		t.Error("title contains no white text pixels")
	}
}

func TestRenderOverlayEmptyText(t *testing.T) {
	pixels, alpha, err := RenderOverlay(Config{Text: "", FrameHeight: 720})

	if err != nil {
		t.Fatalf("Error creating title: %v", err)
	}

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

func TestRenderOverlayFixedWidth(t *testing.T) {
	pixels, alpha, err := RenderOverlay(Config{Text: "Andreas Dashcam", FrameHeight: 720, Width: 1280})
	if err != nil {
		t.Fatalf("Error creating overlay: %v", err)
	}

	if got := pixels.Bounds().Dx(); got != 1280 {
		t.Errorf("brightness width = %d, want 1280", got)
	}
	if got := alpha.Bounds().Dx(); got != 1280 {
		t.Errorf("alpha width = %d, want 1280", got)
	}

	fitted, _, err := RenderOverlay(Config{Text: "Andreas Dashcam", FrameHeight: 720})
	if err != nil {
		t.Fatalf("Error creating fitted overlay: %v", err)
	}
	if pixels.Bounds().Dy() != fitted.Bounds().Dy() {
		t.Errorf("height = %d, want %d like the fitted overlay", pixels.Bounds().Dy(), fitted.Bounds().Dy())
	}
}

func TestRenderOverlayInvalidSize(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"negative width", Config{Text: "A", FrameHeight: 720, Width: -1}},
		{"zero frame height", Config{Text: "A", FrameHeight: 0}},
		{"negative frame height", Config{Text: "A", FrameHeight: -720}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels, alpha, err := RenderOverlay(tt.cfg)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if pixels != nil || alpha != nil {
				t.Error("expected nil images on error")
			}
		})
	}
}
