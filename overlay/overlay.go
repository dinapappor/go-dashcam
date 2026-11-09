package overlay

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
)

type Config struct {
	Title string
}

func RenderTitle(cfg Config) (*image.Gray, *image.Alpha) {
	face := basicfont.Face7x13
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
	return pixels, alpha
}
