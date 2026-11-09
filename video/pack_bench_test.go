package video

import "testing"

// These benchmarks use ordinary Go memory, not camera mappings.
func BenchmarkCopy1080p420(b *testing.B) {
	const width, height = 1920, 1080
	src := make([]byte, width*height*3/2)
	for i := range src {
		src[i] = byte(i)
	}
	rows := func(dst, src []byte, stride, width, height int) []byte {
		for row := 0; row < height; row++ {
			start := row * stride
			dst = append(dst, src[start:start+width]...)
		}
		return dst
	}
	for _, tc := range []struct {
		name      string
		copyPlane func([]byte, []byte, int, int, int) []byte
	}{
		{"rows", rows},
		{"contiguous_planes", appendPlane},
	} {
		b.Run(tc.name, func(b *testing.B) {
			dst := make([]byte, 0, len(src))
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dst = tc.copyPlane(dst[:0], src[:width*height], width, width, height)
				dst = tc.copyPlane(dst, src[width*height:width*height*5/4], width/2, width/2, height/2)
				dst = tc.copyPlane(dst, src[width*height*5/4:], width/2, width/2, height/2)
			}
		})
	}
}
