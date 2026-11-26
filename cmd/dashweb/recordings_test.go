package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setLocal(t *testing.T, zone *time.Location) {
	t.Helper()
	old := time.Local
	time.Local = zone
	t.Cleanup(func() { time.Local = old })
}

func writeRecording(t *testing.T, dir, name string, size int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0600); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, handler, http.MethodGet, path, login(t, handler), nil)
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{bytes: 0, want: "0 B"},
		{bytes: 1, want: "1 B"},
		{bytes: 1023, want: "1023 B"},
		{bytes: 1024, want: "1.0 KiB"},
		{bytes: 1536, want: "1.5 KiB"},
		{bytes: 1<<20 - 1, want: "1024.0 KiB"},
		{bytes: 1 << 20, want: "1.0 MiB"},
		{bytes: 12_500_000, want: "11.9 MiB"},
		{bytes: 3 << 30, want: "3.0 GiB"},
		{bytes: 1<<40 - 1, want: "1024.0 GiB"},
		{bytes: 2 << 40, want: "2.0 TiB"},
		{bytes: 3 << 50, want: "3.0 PiB"},
		{bytes: 1 << 60, want: "1024.0 PiB"},
		{bytes: 1<<63 - 1, want: "8192.0 PiB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatSize(tt.bytes); got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestListRecordingInfo(t *testing.T) {
	zone := time.FixedZone("test", 2*60*60)
	setLocal(t, zone)

	dir := t.TempDir()
	writeRecording(t, dir, "dashcam_20261123-090000.mp4", 100)
	writeRecording(t, dir, "dashcam_20261123-110000.mp4", 300)
	writeRecording(t, dir, "dashcam_20261123-100000.mp4", 200)
	writeRecording(t, dir, "notes.txt", 5)
	writeRecording(t, dir, "dashcam_20261123-120000.mkv", 5)

	got, err := listRecordingInfo(dir)
	if err != nil {
		t.Fatalf("listRecordingInfo: %v", err)
	}

	want := []recordingInfo{
		{Name: "dashcam_20261123-110000.mp4", Size: 300, Time: time.Date(2026, 11, 23, 11, 0, 0, 0, zone), Active: true},
		{Name: "dashcam_20261123-100000.mp4", Size: 200, Time: time.Date(2026, 11, 23, 10, 0, 0, 0, zone)},
		{Name: "dashcam_20261123-090000.mp4", Size: 100, Time: time.Date(2026, 11, 23, 9, 0, 0, 0, zone)},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d recordings, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name {
			t.Errorf("[%d] Name = %q, want %q", i, got[i].Name, want[i].Name)
		}
		if got[i].Size != want[i].Size {
			t.Errorf("[%d] Size = %d, want %d", i, got[i].Size, want[i].Size)
		}
		if !got[i].Time.Equal(want[i].Time) {
			t.Errorf("[%d] Time = %v, want %v", i, got[i].Time, want[i].Time)
		}
		if got[i].Active != want[i].Active {
			t.Errorf("[%d] Active = %t, want %t", i, got[i].Active, want[i].Active)
		}
	}
}

func TestListRecordingInfoBadTimestamp(t *testing.T) {
	dir := t.TempDir()
	writeRecording(t, dir, "dashcam_garbage.mp4", 10)

	got, err := listRecordingInfo(dir)
	if err != nil {
		t.Fatalf("listRecordingInfo: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d recordings, want 1", len(got))
	}
	if !got[0].Time.IsZero() {
		t.Errorf("Time = %v, want zero time", got[0].Time)
	}
	if got[0].Size != 10 {
		t.Errorf("Size = %d, want 10", got[0].Size)
	}
}

func TestListRecordingInfoEmpty(t *testing.T) {
	got, err := listRecordingInfo(t.TempDir())
	if err != nil {
		t.Fatalf("listRecordingInfo: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want no recordings", got)
	}
}

func TestListRecordingInfoMissingDirectory(t *testing.T) {
	got, err := listRecordingInfo(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatalf("expected error, got %+v", got)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestRecordingsPage(t *testing.T) {
	setLocal(t, time.UTC)

	dir := t.TempDir()
	writeRecording(t, dir, "dashcam_20261123-090000.mp4", 1536)
	writeRecording(t, dir, "dashcam_20261123-100000.mp4", 2048)

	recorder := get(t, newHandler(testConfig(dir)), "/recordings")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"dashcam_20261123-090000.mp4",
		"dashcam_20261123-100000.mp4",
		"1.5 KiB",
		"2.0 KiB",
		"2026-11-23 09:00:00",
		"2026-11-23 10:00:00",
		"Recording now",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q:\n%s", want, body)
		}
	}

	newer := strings.Index(body, "dashcam_20261123-100000.mp4")
	older := strings.Index(body, "dashcam_20261123-090000.mp4")
	if newer > older {
		t.Errorf("newest recording is not listed first:\n%s", body)
	}
	if strings.Count(body, "Recording now") != 1 {
		t.Errorf("want exactly one %q:\n%s", "Recording now", body)
	}
	if strings.Contains(body, "<html") {
		t.Errorf("recordings is a fragment and must not contain <html:\n%s", body)
	}
}

func TestRecordingsPageEmpty(t *testing.T) {
	recorder := get(t, newHandler(testConfig(t.TempDir())), "/recordings")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "No recordings") {
		t.Errorf("body does not contain %q:\n%s", "No recordings", recorder.Body.String())
	}
}

func TestRecordingsPageMissingDirectory(t *testing.T) {
	recorder := get(t, newHandler(testConfig(filepath.Join(t.TempDir(), "missing"))), "/recordings")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestFrontPageLoadsRecordings(t *testing.T) {
	recorder := get(t, newHandler(testConfig(t.TempDir())), "/")

	body := recorder.Body.String()
	for _, want := range []string{
		`<script src="/static/htmx.min.js"`,
		`hx-get="/recordings"`,
		`hx-trigger="load, every 10s"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("front page does not contain %q:\n%s", want, body)
		}
	}
}

func TestStaticHTMX(t *testing.T) {
	recorder := send(t, newHandler(testConfig(t.TempDir())), http.MethodGet, "/static/htmx.min.js", nil, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
		t.Errorf("Content-Type = %q, want javascript", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "htmx") {
		t.Error("body does not look like htmx")
	}
}
