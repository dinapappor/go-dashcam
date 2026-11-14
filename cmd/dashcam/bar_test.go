package main

import (
	"testing"
	"time"

	"github.com/dinapappor/go-dashcam/gps"
)

func TestBarText(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.FixedZone("TEST", 2*60*60)
	t.Cleanup(func() { time.Local = oldLocal })

	fixTime := time.Date(2026, 11, 13, 22, 30, 5, 0, time.UTC)

	tests := []struct {
		name string
		fix  gps.Fix
		want string
	}{
		{
			name: "no GPS fix",
			fix:  gps.Fix{},
			want: "No GPS fix",
		},
		{
			name: "no GPS fix ignores time and speed",
			fix:  gps.Fix{RMC: gps.RMC{Time: fixTime, SpeedKts: 10}},
			want: "No GPS fix",
		},
		{
			name: "valid fix with zero speed",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, Time: fixTime}},
			want: "0 km/h 2026-11-14 00:30:05",
		},
		{
			name: "converts knots to rounded km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 10, Time: fixTime}},
			want: "19 km/h 2026-11-14 00:30:05",
		},
		{
			name: "rounds speed up",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 0.3, Time: fixTime}},
			want: "1 km/h 2026-11-14 00:30:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := barText(tt.fix)
			if got != tt.want {
				t.Errorf("barText() = %q, want %q", got, tt.want)
			}
		})
	}
}
