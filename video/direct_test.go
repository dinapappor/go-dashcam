package video

import "testing"

func TestDirectYUV420(t *testing.T) {
	for _, tt := range []struct {
		name                                          string
		width, height, yStride, cStride, length, want int
	}{
		{"tight", 4, 2, 4, 2, 12, 12},
		{"trailing bytes excluded", 4, 2, 4, 2, 16, 12},
		{"odd dimensions", 3, 3, 3, 2, 17, 17},
		{"padded luma", 4, 2, 6, 2, 16, 0},
		{"padded chroma", 4, 2, 4, 3, 14, 0},
		{"short luma", 4, 2, 4, 2, 7, 0},
		{"short chroma", 4, 2, 4, 2, 11, 0},
		{"zero width", 0, 2, 0, 0, 0, 0},
		{"zero height", 4, 0, 4, 2, 0, 0},
		{"negative width", -4, 2, -4, 2, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.length)
			got, ok := DirectYUV420(data, tt.width, tt.height, tt.yStride, tt.cStride)
			if ok != (tt.want > 0) || len(got) != tt.want {
				t.Fatalf("got length=%d, ok=%v; want length=%d", len(got), ok, tt.want)
			}
			if ok {
				got[0] = 42
				if data[0] != 42 {
					t.Fatal("returned a copy instead of borrowing the camera buffer")
				}
			} else if got != nil {
				t.Fatal("expected nil when direct delivery is unavailable")
			}
		})
	}
}
