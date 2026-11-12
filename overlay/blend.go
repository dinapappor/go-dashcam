package overlay

import (
	"image"
)

func blendPixel(frame uint8, title uint8, alpha uint8) uint8 {
	// Make these wider
	f := uint16(frame)
	t := uint16(title)
	a := uint16(alpha)

	// I don't understand this yet
	return uint8((t*a + f*(255-a)) / 255)
}

func BlendOverlay(y []byte, stride int, width int, height int, gray *image.Gray, alpha *image.Alpha, offsetX int, offsetY int) {
	titleWidth := gray.Bounds().Dx()
	titleHeight := gray.Bounds().Dy()

	for row := 0; row < titleHeight; row++ {
		for col := 0; col < titleWidth; col++ {
			frameX := offsetX + col
			frameY := offsetY + row
			if frameX >= width || frameY >= height {
				continue
			}
			frameIndex := frameY*stride + frameX
			titleIndex := row*gray.Stride + col
			alphaIndex := row*alpha.Stride + col

			y[frameIndex] = blendPixel(y[frameIndex], gray.Pix[titleIndex], alpha.Pix[alphaIndex])
		}
	}
}
