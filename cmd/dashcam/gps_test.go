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

func assertBarText(t *testing.T, bar *atomic.Pointer[barImage], text string) {
	t.Helper()
	cfg := testBarConfig()
	cfg.Text = text
	wantGray, wantAlpha, err := overlay.RenderOverlay(cfg)
	if err != nil {
		t.Fatalf("render expected bar: %v", err)
	}
	got := bar.Load()
	if got == nil {
		t.Fatal("bar is nil")
	}
	if !bytes.Equal(got.gray.Pix, wantGray.Pix) || !bytes.Equal(got.alpha.Pix, wantAlpha.Pix) {
		t.Errorf("bar does not show %q", text)
	}
}

func useUTC(t *testing.T) {
	t.Helper()
	oldLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = oldLocal })
}

func TestReadGPSUpdatesBar(t *testing.T) {
	useUTC(t)

	validRMC := nmeaLine("GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W")
	noFixRMC := nmeaLine("GPRMC,123520,V,,,,,,,230394,,,N")
	gga := nmeaLine("GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,")

	tests := []struct {
		name  string
		input string
		unit  string
		want  string
	}{
		{name: "valid fix in km/h", input: validRMC, unit: "km/h", want: " 41 km/h  1994-03-23 12:35:19  N48.1173 E011.5167"},
		{name: "valid fix in mph", input: validRMC, unit: "mph", want: " 26 mph  1994-03-23 12:35:19  N48.1173 E011.5167"},
		{name: "fix lost", input: validRMC + noFixRMC, unit: "km/h", want: "--- km/h  ---------- --:--:--  No GPS fix"},
		{name: "GGA alone is not a fix", input: gga, unit: "km/h", want: "--- km/h  ---------- --:--:--  No GPS fix"},
		{name: "padding and unsupported sentences are skipped", input: "\n\n" + nmeaLine("GPGSV,1,1,00") + validRMC + "\n", unit: "km/h", want: " 41 km/h  1994-03-23 12:35:19  N48.1173 E011.5167"},
		{name: "last fix wins", input: noFixRMC + validRMC, unit: "km/h", want: " 41 km/h  1994-03-23 12:35:19  N48.1173 E011.5167"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := startBar(t, "start")
			r := io.NopCloser(strings.NewReader(tt.input))

			err := readGPS(context.Background(), r, bar, testBarConfig(), tt.unit)
			if err != nil {
				t.Fatalf("readGPS: %v", err)
			}
			assertBarText(t, bar, tt.want)
		})
	}
}

func TestReadGPSEmptyInputKeepsBar(t *testing.T) {
	bar := startBar(t, "start")
	before := bar.Load()

	err := readGPS(context.Background(), io.NopCloser(strings.NewReader("")), bar, testBarConfig(), "km/h")
	if err != nil {
		t.Fatalf("readGPS: %v", err)
	}
	if bar.Load() != before {
		t.Error("bar changed without any GPS sentences")
	}
}

func TestReadGPSReaderError(t *testing.T) {
	readErr := errors.New("bus gone")
	bar := startBar(t, "start")

	err := readGPS(context.Background(), io.NopCloser(iotest.ErrReader(readErr)), bar, testBarConfig(), "km/h")
	if !errors.Is(err, readErr) {
		t.Fatalf("readGPS error = %v, want wrapped %v", err, readErr)
	}
}

func TestReadGPSStopsOnCancel(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	bar := startBar(t, "start")

	done := make(chan error, 1)
	go func() {
		done <- readGPS(ctx, pr, bar, testBarConfig(), "km/h")
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("readGPS after cancel = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("readGPS did not return after cancel")
	}
}
