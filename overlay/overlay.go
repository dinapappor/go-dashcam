package overlay

import (
	"fmt"
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Config struct {
	Text        string
	FrameHeight int // to calculate box height and font size
	Width       int // 0 by default, does not get set for dashcam title
}

func RenderOverlay(cfg Config) (*image.Gray, *image.Alpha, error) {
	if cfg.Width < 0 {
		return nil, nil, fmt.Errorf("Width can not be negative, got: %d", cfg.Width)
	}
	if cfg.FrameHeight <= 0 {
		return nil, nil, fmt.Errorf("Cant calculate font size without frame height")
	}
	// Calculate font size from frame height
	fontSize := float64(cfg.FrameHeight) * 0.02

	gomonofont, err := opentype.Parse(gomono.TTF)
	if err != nil {
		// we return nil because image.gray and alpha are pointers
		return nil, nil, fmt.Errorf("Error loading font: %w", err)
	}
	face, err := opentype.NewFace(
		gomonofont,
		&opentype.FaceOptions{
			Size: fontSize,
			DPI:  72,
		},
	)

	if err != nil {
		return nil, nil, fmt.Errorf("Unable to create new face: %w", err)
	}

	textWidth := font.MeasureString(face, cfg.Text).Ceil()

	// metrics holds the fonts measurements
	metrics := face.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()

	boxWidth := textWidth + 10
	if cfg.Width > 0 {
		boxWidth = cfg.Width
	}
	bounds := image.Rect(0, 0, boxWidth, textHeight+5)

	pixels := image.NewGray(bounds)
	alpha := image.NewAlpha(bounds)

	// set opacity to 128 (~50%) for all pixels
	for i := range alpha.Pix {
		alpha.Pix[i] = 128
	}

	textDrawer := font.Drawer{
		Dst:  pixels, // Draw onto the pixels we created above
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(5, 2+metrics.Ascent.Ceil()),
	}

	textDrawer.DrawString(cfg.Text)

	// Draw it again on the alpha channel so
	// we make the text fully opaque
	textDrawer = font.Drawer{
		Dst:  alpha,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(5, 2+metrics.Ascent.Ceil()),
	}

	textDrawer.DrawString(cfg.Text)
	return pixels, alpha, nil
}
