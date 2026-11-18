package gps

import (
	"testing"
	"time"
)

func TestFixUpdate(t *testing.T) {
	oldRMC := RMC{Valid: true, SpeedKts: 3, Bearing: 45}
	oldGGA := GGA{FixQuality: FIX_VALID, Satellites: 4, HDOP: 2, Altitude: 10}
	newRMC := RMC{
		Time:  time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC),
		Valid: true, Latitude: 48.1, Longitude: 11.5, SpeedKts: 12, Bearing: 90,
	}
	newGGA := GGA{FixQuality: DIFFERENTIAL_GPS_FIX, Satellites: 9, HDOP: 0.8, Altitude: 42}
	initial := Fix{RMC: oldRMC, GGA: oldGGA}

	tests := []struct {
		name    string
		initial Fix
		input   Sentence
		want    Fix
	}{
		{
			name:    "RMC replaces old RMC and preserves GGA",
			initial: initial,
			input:   newRMC,
			want:    Fix{RMC: newRMC, GGA: oldGGA},
		},
		{
			name:    "GGA replaces old GGA and preserves RMC",
			initial: initial,
			input:   newGGA,
			want:    Fix{RMC: oldRMC, GGA: newGGA},
		},
		{
			name:    "invalid RMC clears previous valid RMC",
			initial: initial,
			input:   RMC{Valid: false},
			want:    Fix{GGA: oldGGA},
		},
		{
			name:    "invalid GGA clears previous valid GGA",
			initial: initial,
			input:   GGA{FixQuality: FIX_NOT_VALID},
			want:    Fix{RMC: oldRMC},
		},
		{
			name:  "first RMC initializes empty fix",
			input: newRMC,
			want:  Fix{RMC: newRMC},
		},
		{
			name:  "first GGA initializes empty fix",
			input: newGGA,
			want:  Fix{GGA: newGGA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := tt.initial
			fix.Update(tt.input)
			if fix != tt.want {
				t.Errorf("Fix = %+v, want %+v", fix, tt.want)
			}
		})
	}
}

func TestFixTrusted(t *testing.T) {
	good := func(quality FixQuality, hdop float64) Fix {
		return Fix{RMC: RMC{Valid: true}, GGA: GGA{FixQuality: quality, Satellites: 8, HDOP: hdop}}
	}

	tests := []struct {
		name string
		fix  Fix
		want bool
	}{
		{name: "empty fix", fix: Fix{}, want: false},
		{name: "GPS fix", fix: good(FIX_VALID, 1.2), want: true},
		{name: "DGPS fix", fix: good(DIFFERENTIAL_GPS_FIX, 1.2), want: true},
		{name: "PPS fix", fix: good(PPS_FIX, 1.2), want: true},
		{name: "RTK fixed", fix: good(RTK_FIXED, 1.2), want: true},
		{name: "RTK float", fix: good(RTK_FLOAT, 1.2), want: true},
		{name: "no fix quality", fix: good(FIX_NOT_VALID, 1.2), want: false},
		{name: "dead reckoning", fix: good(INS, 1.2), want: false},
		{name: "manual", fix: good(MANUAL, 1.2), want: false},
		{name: "simulation", fix: good(SIMULATION, 1.2), want: false},
		{name: "unknown fix quality", fix: good(FixQuality(42), 1.2), want: false},
		{name: "RMC invalid", fix: Fix{RMC: RMC{Valid: false}, GGA: GGA{FixQuality: FIX_VALID, HDOP: 1.2}}, want: false},
		{name: "RMC without GGA", fix: Fix{RMC: RMC{Valid: true}}, want: false},
		{name: "HDOP just below limit", fix: good(FIX_VALID, 4.99), want: true},
		{name: "HDOP at limit", fix: good(FIX_VALID, maxHDOP), want: false},
		{name: "HDOP above limit", fix: good(FIX_VALID, 12.5), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fix.Trusted(); got != tt.want {
				t.Errorf("Trusted() = %v, want %v", got, tt.want)
			}
		})
	}
}
