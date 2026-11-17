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

	now := time.Date(2026, 11, 16, 12, 35, 0, 0, time.UTC)

	tests := []struct {
		name   string
		fix    gps.Fix
		unit   string
		now    time.Time
		synced bool
		want   string
	}{
		{
			name: "no GPS fix",
			unit: "km/h",
			fix:  gps.Fix{},
			want: "--- km/h  ---------- --:--:--  No GPS fix",
		},
		{
			name: "no GPS fix ignores time, speed and position",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Time: fixTime, SpeedKts: 10, Latitude: 59.3293, Longitude: 18.0686}},
			want: "--- km/h  ---------- --:--:--  No GPS fix",
		},
		{
			name: "no GPS fix in mph",
			unit: "mph",
			fix:  gps.Fix{RMC: gps.RMC{Time: fixTime, SpeedKts: 10}},
			want: "--- mph  ---------- --:--:--  No GPS fix",
		},
		{
			name: "valid fix with zero speed",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			want: "  0 km/h  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
		{
			name: "converts knots to rounded km/h",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 10, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			want: " 19 km/h  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
		{
			name: "three digit speed fills the slot",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 76.2, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			want: "141 km/h  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
		{
			name: "rounds speed up",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 0.3, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			want: "  1 km/h  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
		{
			name: "converts knots to rounded mph",
			unit: "mph",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 10, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			want: " 12 mph  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
		{
			name: "southern and western hemispheres",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, Time: fixTime, Latitude: -33.8688, Longitude: -151.2093}},
			want: "  0 km/h  2026-11-14 00:30:05  S33.8688 W151.2093",
		},
		{
			name: "small coordinates are zero padded",
			unit: "km/h",
			fix:  gps.Fix{RMC: gps.RMC{Valid: true, Time: fixTime, Latitude: 9.5, Longitude: 4.25}},
			want: "  0 km/h  2026-11-14 00:30:05  N09.5000 E004.2500",
		},
		{
			name:   "no GPS fix with synced clock shows local clock time",
			unit:   "km/h",
			fix:    gps.Fix{},
			now:    now,
			synced: true,
			want:   "--- km/h  2026-11-16 14:35:00  No GPS fix",
		},
		{
			name:   "no GPS fix with synced clock in mph",
			unit:   "mph",
			fix:    gps.Fix{},
			now:    now,
			synced: true,
			want:   "--- mph  2026-11-16 14:35:00  No GPS fix",
		},
		{
			name:   "no GPS fix with unsynced clock hides time",
			unit:   "km/h",
			fix:    gps.Fix{},
			now:    now,
			synced: false,
			want:   "--- km/h  ---------- --:--:--  No GPS fix",
		},
		{
			name:   "no GPS fix ignores GPS time when clock is synced",
			unit:   "km/h",
			fix:    gps.Fix{RMC: gps.RMC{Time: fixTime, SpeedKts: 10}},
			now:    now,
			synced: true,
			want:   "--- km/h  2026-11-16 14:35:00  No GPS fix",
		},
		{
			name:   "valid fix uses GPS time, not clock time",
			unit:   "km/h",
			fix:    gps.Fix{RMC: gps.RMC{Valid: true, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}},
			now:    now,
			synced: true,
			want:   "  0 km/h  2026-11-14 00:30:05  N59.3293 E018.0686",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := barText(tt.fix, tt.unit, tt.now, tt.synced)
			if got != tt.want {
				t.Errorf("barText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBarTextKeepsColumns(t *testing.T) {
	fixTime := time.Date(2026, 11, 13, 22, 30, 5, 0, time.UTC)
	now := time.Date(2026, 11, 16, 12, 35, 0, 0, time.UTC)
	withFix := barText(gps.Fix{RMC: gps.RMC{Valid: true, SpeedKts: 76.2, Time: fixTime, Latitude: 59.3293, Longitude: 18.0686}}, "km/h", now, false)

	for _, synced := range []bool{false, true} {
		noFix := barText(gps.Fix{}, "km/h", now, synced)

		timeColumn := len("141 km/h  ")
		positionColumn := len("141 km/h  2026-11-14 00:30:05  ")

		if withFix[timeColumn-2:timeColumn] != "  " || noFix[timeColumn-2:timeColumn] != "  " {
			t.Errorf("synced=%v: time column differs:\n%q\n%q", synced, withFix, noFix)
		}
		if withFix[positionColumn-2:positionColumn] != "  " || noFix[positionColumn-2:positionColumn] != "  " {
			t.Errorf("synced=%v: position column differs:\n%q\n%q", synced, withFix, noFix)
		}
	}
}

func TestFormatCoordinate(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		positive string
		negative string
		width    int
		want     string
	}{
		{name: "north", value: 59.3293, positive: "N", negative: "S", width: 7, want: "N59.3293"},
		{name: "south", value: -33.8688, positive: "N", negative: "S", width: 7, want: "S33.8688"},
		{name: "east with three digit padding", value: 18.0686, positive: "E", negative: "W", width: 8, want: "E018.0686"},
		{name: "west three digits", value: -151.2093, positive: "E", negative: "W", width: 8, want: "W151.2093"},
		{name: "single digit latitude", value: 9.5, positive: "N", negative: "S", width: 7, want: "N09.5000"},
		{name: "zero counts as positive", value: 0, positive: "N", negative: "S", width: 7, want: "N00.0000"},
		{name: "rounds to four decimals", value: 59.32938, positive: "N", negative: "S", width: 7, want: "N59.3294"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCoordinate(tt.value, tt.positive, tt.negative, tt.width)
			if got != tt.want {
				t.Errorf("formatCoordinate(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
