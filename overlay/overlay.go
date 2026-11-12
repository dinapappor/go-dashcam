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
	Title       string
	FrameHeight int // to calculate box height and font size
}

func RenderTitle(cfg Config) (*image.Gray, *image.Alpha, error) {
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

	textWidth := font.MeasureString(face, cfg.Title).Ceil()

	// metrics holds the fonts measurements
	metrics := face.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()

	bounds := image.Rect(0, 0, textWidth+10, textHeight+5)

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

	textDrawer.DrawString(cfg.Title)

	// Draw it again on the alpha channel so
	// we make the text fully opaque
	textDrawer = font.Drawer{
		Dst:  alpha,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(5, 2+metrics.Ascent.Ceil()),
	}

	textDrawer.DrawString(cfg.Title)
	return pixels, alpha, nil
}
