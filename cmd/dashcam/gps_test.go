package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/dinapappor/go-dashcam/gps"
	"github.com/dinapappor/go-dashcam/overlay"
)

func nmeaLine(body string) string {
	return fmt.Sprintf("$%s*%02X\r\n", body, gps.CalculateChecksum(body))
}

func testBarConfig() overlay.Config {
	return overlay.Config{FrameHeight: 720, Width: 1280}
}

func startBar(t *testing.T, text string) *atomic.Pointer[barImage] {
	t.Helper()
	cfg := testBarConfig()
	cfg.Text = text
	g, a, err := overlay.RenderOverlay(cfg)
	if err != nil {
		t.Fatalf("render start bar: %v", err)
	}
	var bar atomic.Pointer[barImage]
	bar.Store(&barImage{gray: g, alpha: a})
	return &bar
}

func barShows(bar *atomic.Pointer[barImage], text string) bool {
	cfg := testBarConfig()
	cfg.Text = text
	wantGray, wantAlpha, err := overlay.RenderOverlay(cfg)
	if err != nil {
		return false
	}
	got := bar.Load()
	return got != nil && bytes.Equal(got.gray.Pix, wantGray.Pix) && bytes.Equal(got.alpha.Pix, wantAlpha.Pix)
}

func waitForBarText(t *testing.T, bar *atomic.Pointer[barImage], text string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if barShows(bar, text) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("bar does not show %q after %v", text, timeout)
}

func useUTC(t *testing.T) {
	t.Helper()
	oldLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = oldLocal })
}

func fakeBarClock(t *testing.T, now time.Time, synced bool) {
	t.Helper()
	oldNow, oldSynced := barNow, barClockSynced
	barNow = func() time.Time { return now }
	barClockSynced = func() bool { return synced }
	t.Cleanup(func() { barNow, barClockSynced = oldNow, oldSynced })
}

func readAllFixes(t *testing.T, input string) []gps.Fix {
	t.Helper()
	fixes := make(chan gps.Fix, 32)
	err := readGPS(context.Background(), io.NopCloser(strings.NewReader(input)), fixes)
	if err != nil {
		t.Fatalf("readGPS: %v", err)
	}
	close(fixes)
	var got []gps.Fix
	for fix := range fixes {
		got = append(got, fix)
	}
	return got
}

func TestReadGPSSendsFixes(t *testing.T) {
	validRMC := nmeaLine("GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W")
	noFixRMC := nmeaLine("GPRMC,123520,V,,,,,,,230394,,,N")
	gga := nmeaLine("GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,")

	tests := []struct {
		name           string
		input          string
		wantCount      int
		wantValid      bool
		wantSpeedKts   float64
		wantSatellites int
	}{
		{name: "valid RMC", input: validRMC, wantCount: 1, wantValid: true, wantSpeedKts: 22.4},
		{name: "fix lost", input: validRMC + noFixRMC, wantCount: 2, wantValid: false},
		{name: "GGA alone is not a fix", input: gga, wantCount: 1, wantValid: false, wantSatellites: 8},
		{name: "RMC then GGA keeps both", input: validRMC + gga, wantCount: 2, wantValid: true, wantSpeedKts: 22.4, wantSatellites: 8},
		{name: "padding, unsupported and bad sentences are skipped", input: "\n\n" + nmeaLine("GPGSV,1,1,00") + "$GPRMC,bad*00\r\n" + validRMC + "\n", wantCount: 1, wantValid: true, wantSpeedKts: 22.4},
		{name: "last fix wins", input: noFixRMC + validRMC, wantCount: 2, wantValid: true, wantSpeedKts: 22.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readAllFixes(t, tt.input)
			if len(got) != tt.wantCount {
				t.Fatalf("got %d fixes, want %d", len(got), tt.wantCount)
			}
			last := got[len(got)-1]
			if last.RMC.Valid != tt.wantValid {
				t.Errorf("RMC.Valid = %v, want %v", last.RMC.Valid, tt.wantValid)
			}
			if last.RMC.SpeedKts != tt.wantSpeedKts {
				t.Errorf("RMC.SpeedKts = %v, want %v", last.RMC.SpeedKts, tt.wantSpeedKts)
			}
			if last.GGA.Satellites != tt.wantSatellites {
				t.Errorf("GGA.Satellites = %v, want %v", last.GGA.Satellites, tt.wantSatellites)
			}
		})
	}
}

func TestReadGPSEmptyInputSendsNothing(t *testing.T) {
	if got := readAllFixes(t, ""); len(got) != 0 {
		t.Errorf("got %d fixes, want 0", len(got))
	}
}

func TestReadGPSReaderError(t *testing.T) {
	readErr := errors.New("bus gone")
	fixes := make(chan gps.Fix, 1)

	err := readGPS(context.Background(), io.NopCloser(iotest.ErrReader(readErr)), fixes)
	if !errors.Is(err, readErr) {
		t.Fatalf("readGPS error = %v, want wrapped %v", err, readErr)
	}
}

func waitForReturn(t *testing.T, done <-chan error, what string) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s after cancel = %v, want nil", what, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s did not return after cancel", what)
	}
}

func TestReadGPSStopsOnCancelWhileReading(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- readGPS(ctx, pr, make(chan gps.Fix))
	}()

	cancel()
	waitForReturn(t, done, "readGPS")
}

func TestReadGPSStopsOnCancelWhileSending(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	unread := make(chan gps.Fix)
	done := make(chan error, 1)
	go func() {
		done <- readGPS(ctx, pr, unread)
	}()

	pw.Write([]byte(nmeaLine("GPRMC,123520,V,,,,,,,230394,,,N")))
	time.Sleep(20 * time.Millisecond)
	cancel()
	waitForReturn(t, done, "readGPS")
}

func runBarLoop(t *testing.T, fixes <-chan gps.Fix, unit string) (*atomic.Pointer[barImage], context.CancelFunc, <-chan struct{}) {
	t.Helper()
	bar := startBar(t, "start")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		barLoop(ctx, bar, testBarConfig(), unit, fixes)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return bar, cancel, done
}

func TestBarLoopDrawsFixes(t *testing.T) {
	useUTC(t)
	fakeBarClock(t, time.Date(2026, 11, 16, 12, 35, 0, 0, time.UTC), true)

	valid := gps.Fix{GGA: trustedGGA, RMC: gps.RMC{Valid: true, SpeedKts: 22.4, Time: time.Date(1994, 3, 23, 12, 35, 19, 0, time.UTC), Latitude: 48.1173, Longitude: 11.516667}}

	tests := []struct {
		name  string
		fixes []gps.Fix
		unit  string
		want  string
	}{
		{name: "valid fix in km/h", fixes: []gps.Fix{valid}, unit: "km/h", want: " 41 km/h  1994-03-23 12:35:19  N48.1173 E011.5167  ▁▃▅ "},
		{name: "valid fix in mph", fixes: []gps.Fix{valid}, unit: "mph", want: " 26 mph  1994-03-23 12:35:19  N48.1173 E011.5167  ▁▃▅ "},
		{name: "fix lost shows clock time", fixes: []gps.Fix{valid, {}}, unit: "km/h", want: "--- km/h  2026-11-16 12:35:00  No GPS fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixes := make(chan gps.Fix)
			bar, _, _ := runBarLoop(t, fixes, tt.unit)
			for _, fix := range tt.fixes {
				fixes <- fix
			}
			waitForBarText(t, bar, tt.want, 500*time.Millisecond)
		})
	}
}

func TestBarLoopTicksWithoutGPS(t *testing.T) {
	useUTC(t)

	tests := []struct {
		name   string
		synced bool
		want   string
	}{
		{name: "synced clock shows time", synced: true, want: "--- km/h  2026-11-16 12:35:00  No GPS fix"},
		{name: "unsynced clock shows dashes", synced: false, want: "--- km/h  ---------- --:--:--  No GPS fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeBarClock(t, time.Date(2026, 11, 16, 12, 35, 0, 0, time.UTC), tt.synced)
			bar, _, _ := runBarLoop(t, make(chan gps.Fix), "km/h")
			waitForBarText(t, bar, tt.want, 2*time.Second)
		})
	}
}

func movableBarClock(t *testing.T, start time.Time) func(time.Duration) {
	t.Helper()
	var now atomic.Pointer[time.Time]
	now.Store(&start)
	oldNow, oldSynced := barNow, barClockSynced
	barNow = func() time.Time { return *now.Load() }
	barClockSynced = func() bool { return true }
	t.Cleanup(func() { barNow, barClockSynced = oldNow, oldSynced })
	return func(d time.Duration) {
		later := now.Load().Add(d)
		now.Store(&later)
	}
}

func TestBarLoopFixAge(t *testing.T) {
	useUTC(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	valid := gps.Fix{GGA: trustedGGA, RMC: gps.RMC{Valid: true, SpeedKts: 22.4, Time: time.Date(1994, 3, 23, 12, 35, 19, 0, time.UTC), Latitude: 48.1173, Longitude: 11.516667}}
	validText := " 41 km/h  1994-03-23 12:35:19  N48.1173 E011.5167  ▁▃▅ "

	tests := []struct {
		name    string
		advance time.Duration
		want    string
	}{
		{name: "fresh fix is kept", advance: 2 * time.Second, want: validText},
		{name: "fix at max age is kept", advance: maxFixAge, want: validText},
		{name: "stale fix shows no fix", advance: 4 * time.Second, want: "--- km/h  2026-11-18 12:00:04  No GPS fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance := movableBarClock(t, start)
			fixes := make(chan gps.Fix)
			bar, _, _ := runBarLoop(t, fixes, "km/h")

			fixes <- valid
			waitForBarText(t, bar, validText, 500*time.Millisecond)

			advance(tt.advance)
			time.Sleep(1200 * time.Millisecond)
			if !barShows(bar, tt.want) {
				t.Errorf("after %v bar does not show %q", tt.advance, tt.want)
			}
		})
	}
}

func TestBarLoopStopsOnCancel(t *testing.T) {
	_, cancel, done := runBarLoop(t, make(chan gps.Fix), "km/h")
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("barLoop did not return after cancel")
	}
}
