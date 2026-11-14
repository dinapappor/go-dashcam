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
