package video

// DirectYUV420 returns a borrowed view of tightly packed Y, Cb, Cr data.
// The caller must finish using it before returning the buffer to the camera.
// Padded or incomplete frames must use another path.
func DirectYUV420(data []byte, width, height, yStride, cStride int) ([]byte, bool) {
	if width <= 0 || height <= 0 || yStride != width {
		return nil, false
	}
	chromaWidth := width/2 + width%2
	chromaHeight := height/2 + height%2
	if cStride != chromaWidth || width > len(data)/height {
		return nil, false
	}
	ySize := width * height
	if chromaWidth > (len(data)-ySize)/2/chromaHeight {
		return nil, false
	}
	size := ySize + 2*chromaWidth*chromaHeight
	return data[:size], true
}
