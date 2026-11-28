package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUsedPercent(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		free  int64
		want  int
	}{
		{name: "empty disk", total: 1000, free: 1000, want: 0},
		{name: "full disk", total: 1000, free: 0, want: 100},
		{name: "half full", total: 1000, free: 500, want: 50},
		{name: "rounds down", total: 1000, free: 1, want: 99},
		{name: "a little used", total: 1000, free: 999, want: 0},
		{name: "one third used", total: 3, free: 2, want: 33},
		{name: "large disk", total: 2 << 40, free: 1 << 40, want: 50},
		{name: "zero total", total: 0, free: 0, want: 0},
		{name: "free larger than total", total: 100, free: 200, want: 0},
		{name: "negative free", total: 100, free: -1, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usedPercent(tt.total, tt.free); got != tt.want {
				t.Errorf("usedPercent(%d, %d) = %d, want %d", tt.total, tt.free, got, tt.want)
			}
		})
	}
}

func TestReadStats(t *testing.T) {
	zone := time.FixedZone("test", 2*60*60)
	setLocal(t, zone)

	dir := t.TempDir()
	writeRecording(t, dir, "dashcam_20261123-100000.mp4", 2048)
	writeRecording(t, dir, "dashcam_20261123-090000.mp4", 1536)
	writeRecording(t, dir, "dashcam_20261123-110000.mp4", 512)
	writeRecording(t, dir, "notes.txt", 5000)

	got, err := readStats(dir)
	if err != nil {
		t.Fatalf("readStats: %v", err)
	}

	if got.Count != 3 {
		t.Errorf("Count = %d, want 3", got.Count)
	}
	if got.Used != 4096 {
		t.Errorf("Used = %d, want 4096", got.Used)
	}
	if want := time.Date(2026, 11, 23, 9, 0, 0, 0, zone); !got.Oldest.Equal(want) {
		t.Errorf("Oldest = %v, want %v", got.Oldest, want)
	}
	if got.Total <= 0 {
		t.Errorf("Total = %d, want more than zero", got.Total)
	}
	if got.Free < 0 || got.Free > got.Total {
		t.Errorf("Free = %d, want between 0 and Total %d", got.Free, got.Total)
	}
	if want := usedPercent(got.Total, got.Free); got.UsedPercent != want {
		t.Errorf("UsedPercent = %d, want usedPercent(Total, Free) = %d", got.UsedPercent, want)
	}
}

func TestReadStatsEmpty(t *testing.T) {
	got, err := readStats(t.TempDir())
	if err != nil {
		t.Fatalf("readStats: %v", err)
	}
	if got.Count != 0 {
		t.Errorf("Count = %d, want 0", got.Count)
	}
	if got.Used != 0 {
		t.Errorf("Used = %d, want 0", got.Used)
	}
	if !got.Oldest.IsZero() {
		t.Errorf("Oldest = %v, want zero time", got.Oldest)
	}
	if got.Total <= 0 {
		t.Errorf("Total = %d, want more than zero", got.Total)
	}
}

func TestReadStatsMissingDirectory(t *testing.T) {
	if _, err := readStats(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

func progressValue(t *testing.T, body string) int {
	t.Helper()
	bar, found := tagWith(body, "<div", "progress-bar")
	if !found {
		t.Fatalf("stats has no progress-bar:\n%s", body)
	}
	for _, want := range []string{`role="progressbar"`, `aria-valuemin="0"`, `aria-valuemax="100"`} {
		if !strings.Contains(bar, want) {
			t.Errorf("progress-bar %q does not contain %q", bar, want)
		}
	}
	text, found := attributeValue(bar, "<div", "aria-valuenow")
	if !found {
		t.Fatalf("progress-bar has no aria-valuenow: %q", bar)
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		t.Fatalf("aria-valuenow = %q, want a whole number", text)
	}
	if value < 0 || value > 100 {
		t.Errorf("aria-valuenow = %d, want 0 to 100", value)
	}
	if style, _ := attributeValue(bar, "<div", "style"); !strings.Contains(style, "width: "+text+"%") {
		t.Errorf("progress-bar style = %q, want it to contain %q", style, "width: "+text+"%")
	}
	return value
}

func TestStatsFragment(t *testing.T) {
	setLocal(t, time.UTC)

	dir := t.TempDir()
	writeRecording(t, dir, "dashcam_20261123-090000.mp4", 1536)
	writeRecording(t, dir, "dashcam_20261123-100000.mp4", 2048)

	recorder := get(t, newHandler(testConfig(dir)), "/stats")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"Recordings",
		"Storage used",
		"Free space",
		"Oldest recording",
		">2<",
		"3.5 KiB",
		"2026-11-23 09:00:00",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("stats does not contain %q:\n%s", want, body)
		}
	}
	if strings.Count(body, `class="card"`) != 4 {
		t.Errorf("want 4 Tabler cards:\n%s", body)
	}
	if strings.Contains(body, "2026-11-23 10:00:00") {
		t.Errorf("stats shows the newest recording, want only the oldest:\n%s", body)
	}
	if strings.Contains(body, "<html") {
		t.Errorf("stats is a fragment and must not contain <html:\n%s", body)
	}
	progressValue(t, body)
}

func TestStatsFragmentEmpty(t *testing.T) {
	recorder := get(t, newHandler(testConfig(t.TempDir())), "/stats")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, want := range []string{">0<", "0 B", "None"} {
		if !strings.Contains(body, want) {
			t.Errorf("stats does not contain %q:\n%s", want, body)
		}
	}
	progressValue(t, body)
}

func TestStatsMissingDirectory(t *testing.T) {
	recorder := get(t, newHandler(testConfig(filepath.Join(t.TempDir(), "missing"))), "/stats")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
