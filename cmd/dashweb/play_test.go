package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	finishedRecording = "dashcam_20261123-090000.mp4"
	activeRecording   = "dashcam_20261123-100000.mp4"
)

var finishedContent = []byte("0123456789abcdefghijklmnopqrstuvwxyz")

func playHandler(t *testing.T) http.Handler {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "videos")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, finishedRecording), finishedContent, 0600); err != nil {
		t.Fatal(err)
	}
	writeRecording(t, dir, activeRecording, 64)
	writeRecording(t, dir, "notes.txt", 16)
	writeRecording(t, dir, "dashcam_20261123-080000.mkv", 16)
	writeRecording(t, parent, "dashcam_20261123-070000.mp4", 16)
	return newHandler(testConfig(dir))
}

func getWithRange(t *testing.T, handler http.Handler, path, byteRange string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Range", byteRange)
	request.AddCookie(login(t, handler))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestVideoServesRecording(t *testing.T) {
	recorder := get(t, playHandler(t), "/video/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "video/mp4" {
		t.Errorf("Content-Type = %q, want %q", contentType, "video/mp4")
	}
	if got := recorder.Body.String(); got != string(finishedContent) {
		t.Errorf("body = %q, want %q", got, finishedContent)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); disposition != "" {
		t.Errorf("Content-Disposition = %q, want none so the browser plays it inline", disposition)
	}
}

func TestVideoRange(t *testing.T) {
	recorder := getWithRange(t, playHandler(t), "/video/"+finishedRecording, "bytes=0-9")

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if got, want := recorder.Body.String(), string(finishedContent[:10]); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "video/mp4" {
		t.Errorf("Content-Type = %q, want %q", contentType, "video/mp4")
	}
}

func TestVideoDownload(t *testing.T) {
	recorder := get(t, playHandler(t), "/video/"+finishedRecording+"?download=1")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	disposition := recorder.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment") {
		t.Errorf("Content-Disposition = %q, want it to start with %q", disposition, "attachment")
	}
	if !strings.Contains(disposition, finishedRecording) {
		t.Errorf("Content-Disposition = %q, want it to contain %q", disposition, finishedRecording)
	}
	if got := recorder.Body.String(); got != string(finishedContent) {
		t.Errorf("body = %q, want %q", got, finishedContent)
	}
}

var badRecordingNames = []struct {
	name string
	path string
}{
	{"not a recording", "notes.txt"},
	{"wrong suffix", "dashcam_20261123-080000.mkv"},
	{"missing file", "dashcam_20261123-080000.mp4"},
	{"escapes the directory", "..%2fdashcam_20261123-070000.mp4"},
}

func TestVideoRejectsBadNames(t *testing.T) {
	handler := playHandler(t)
	for _, tt := range badRecordingNames {
		t.Run(tt.name, func(t *testing.T) {
			recorder := get(t, handler, "/video/"+tt.path)
			if recorder.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
		})
	}
}

func TestVideoActiveRecording(t *testing.T) {
	handler := playHandler(t)
	for _, path := range []string{
		"/video/" + activeRecording,
		"/video/" + activeRecording + "?download=1",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := get(t, handler, path)
			if recorder.Code != http.StatusConflict {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
			}
		})
	}
}

func TestPlayPage(t *testing.T) {
	recorder := get(t, playHandler(t), "/play/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"<html",
		"<video",
		"controls",
		"playsinline",
		`src="/video/` + finishedRecording + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("play page does not contain %q:\n%s", want, body)
		}
	}
}

func TestPlayRejectsBadNames(t *testing.T) {
	handler := playHandler(t)
	for _, tt := range badRecordingNames {
		t.Run(tt.name, func(t *testing.T) {
			recorder := get(t, handler, "/play/"+tt.path)
			if recorder.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
		})
	}
}

func TestPlayActiveRecording(t *testing.T) {
	recorder := get(t, playHandler(t), "/play/"+activeRecording)

	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestRecordingsPageLinks(t *testing.T) {
	recorder := get(t, playHandler(t), "/recordings/list")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`href="/play/` + finishedRecording + `"`,
		`href="/video/` + finishedRecording + `?download=1"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("recordings page does not contain %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{
		"/play/" + activeRecording,
		"/video/" + activeRecording,
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("recordings page links the active recording (%q):\n%s", unwanted, body)
		}
	}
}
