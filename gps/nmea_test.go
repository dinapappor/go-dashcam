package gps

import (
	"math"
	"testing"
	"time"
)

func TestValidateSentence(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Test success",
			input:   "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A",
			want:    "GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W",
			wantErr: false,
		},
		{
			name:    "Missing $ fails",
			input:   "GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing asterisk fails",
			input:   "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W6A",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing checksum after * fails",
			input:   "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*",
			want:    "",
			wantErr: true,
		},
		{
			name:    "checksum mismatch",
			input:   "$GPRMC,123519,A,4807.038,N,01131.000,E,022.5,084.4,230394,003.1,W*6A",
			want:    "",
			wantErr: true,
		},
		{
			name:    "incorrect checksum", // Hex doesnt go to z, duh
			input:   "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*ZZ",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSentence(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Expected sentence wrong, expected %s, got: %s", tt.want, got)
			}
		})
	}
}

func TestParseCoordinate(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		hemisphere string
		want       float64
		wantErr    bool
	}{
		{name: "northern hemisphere", value: "4807.038", hemisphere: "N", want: 48.1173, wantErr: false},
		{name: "southern hemisphere", value: "4807.038", hemisphere: "S", want: -48.1173, wantErr: false},
		{name: "eastern hemisphere", value: "01131.000", hemisphere: "E", want: 11.516667, wantErr: false},
		{name: "western hemisphere", value: "01131.000", hemisphere: "W", want: -11.516667, wantErr: false},
		{name: "ROEV hemisphere", value: "01131.000", hemisphere: "ROEV", want: 0, wantErr: true},
		{name: "empty value", value: "", hemisphere: "N", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoordinate(tt.value, tt.hemisphere)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}

			// check difference, if it's more than 10 cm off (1e-6)
			// we fail the test. (even if our gps cant be that accurate hehe)
			if math.Abs(got-tt.want) > 1e-6 {
				t.Fatalf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestParseRMC(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    RMC
		wantErr bool
	}{
		{
			name: "success",
			input: []string{
				"123519",
				"A",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"084.4",
				"230394",
				"003.1",
				"W",
			},
			want: RMC{
				Time:      time.Date(1994, 3, 23, 12, 35, 19, 0, time.UTC),
				Valid:     true,
				Latitude:  48.1173,
				Longitude: 11.516667,
				SpeedKts:  22.4,
				Bearing:   84.4,
			},
			wantErr: false,
		},
		{
			name: "bearing-should-be-zero-when-empty",
			input: []string{
				"123519",
				"A",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"",
				"230394",
				"003.1",
				"W",
			},
			want: RMC{
				Time:      time.Date(1994, 3, 23, 12, 35, 19, 0, time.UTC),
				Valid:     true,
				Latitude:  48.1173,
				Longitude: 11.516667,
				SpeedKts:  22.4,
				Bearing:   0,
			},
			wantErr: false,
		},
		{
			name: "short sentence",
			input: []string{
				"123519",
				"A",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"084.4",
				"230394",
				"003.1",
			},
			wantErr: true,
		},
		{
			name: "fix-validity-incorrect",
			input: []string{
				"123519",
				"ROEV",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"084.4",
				"230394",
				"003.1",
				"W",
			},
			wantErr: true,
		},
		{
			name: "validity-v",
			input: []string{
				"123519",
				"V",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"084.4",
				"230394",
				"003.1",
				"W",
			},
			want:    RMC{Valid: false},
			wantErr: false,
		},
		{
			name: "bad-bearing",
			input: []string{
				"123519",
				"A",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"022.4",
				"roev",
				"230394",
				"003.1",
				"W",
			},
			wantErr: true,
		},
		{
			name: "empty-speed-should-err",
			input: []string{
				"123519",
				"A",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"",
				"084.4",
				"230394",
				"003.1",
				"W",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRMC(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}

			// if we expect an error, there's nothing else to check here
			if tt.wantErr {
				return
			}

			if got.Valid != tt.want.Valid {
				t.Fatalf("Valid check failed, expected: %t, got:%t", tt.want.Valid, got.Valid)
			}

			if !got.Time.Equal(tt.want.Time) {
				t.Fatalf("Did not get correct time, expected: %v, got: %v", tt.want.Time, got.Time)
			}
			// check difference, if it's more than 10 cm off (1e-6)
			// we fail the test. (even if our gps cant be that accurate hehe)
			if math.Abs(got.Latitude-tt.want.Latitude) > 1e-6 {
				t.Errorf("Latitude: got %f, want %f", got.Latitude, tt.want.Latitude)
			}
			if math.Abs(got.Longitude-tt.want.Longitude) > 1e-6 {
				t.Errorf("Longitude: got %f, want %f", got.Longitude, tt.want.Longitude)
			}
			if math.Abs(got.SpeedKts-tt.want.SpeedKts) > 1e-6 {
				t.Errorf("SpeedKts: got %f, want %f", got.SpeedKts, tt.want.SpeedKts)
			}
			if math.Abs(got.Bearing-tt.want.Bearing) > 1e-6 {
				t.Errorf("Bearing: got %f, want %f", got.Bearing, tt.want.Bearing)
			}
		})
	}
}

func TestParseGGA(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    GGA
		wantErr bool
	}{
		{
			name: "success",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"1",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			want: GGA{
				FixQuality: FIX_VALID,
				Satellites: 8,
				HDOP:       0.9,
				Altitude:   545.4,
			},
			wantErr: false,
		},
		{
			name: "no-fix-no-error",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"0",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			want: GGA{
				FixQuality: FIX_NOT_VALID,
			},
			wantErr: false,
		},
		{
			name: "too few fields",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"1",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
			},
			wantErr: true,
		},
		{
			name: "quality-out-of-bounds-upwards",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"9",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "quality-out-of-bounds-negative",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"-1",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "quality-empty",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"",
				"08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "sattelites-negative",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"-1",
				"-08",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "sattelites-empty",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"-1",
				"",
				"0.9",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "hdop-bad",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"-1",
				"",
				"1",
				"545.4",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			wantErr: true,
		},
		{
			name: "altitude-bad",
			input: []string{
				"123519",
				"4807.038",
				"N",
				"01131.000",
				"E",
				"1",
				"08",
				"0.9",
				"roev",
				"M",
				"46.9",
				"M",
				"",
				"",
			},
			want: GGA{
				FixQuality: FIX_VALID,
				Satellites: 8,
				HDOP:       0.9,
				Altitude:   545.4,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGGA(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}

			// if we expect an error, there's nothing else to check here
			if tt.wantErr {
				return
			}

			if got.FixQuality != tt.want.FixQuality {
				t.Fatalf("FixQuality check failed, expected: %d, got:%d", tt.want.FixQuality, got.FixQuality)
			}

			if got.Satellites != tt.want.Satellites {
				t.Fatalf("Wrong number of sattelites was returned, expected: %d, got: %d", tt.want.Satellites, got.Satellites)
			}

			if math.Abs(got.HDOP-tt.want.HDOP) > 1e-6 {
				t.Errorf("HDOP mismatch, expected: %f, got: %f", tt.want.HDOP, got.HDOP)
			}
			if math.Abs(got.Altitude-tt.want.Altitude) > 1e-6 {
				t.Errorf("Altitude mismatch, expected: %f, got: %f", tt.want.Altitude, got.Altitude)
			}
		})
	}
}
