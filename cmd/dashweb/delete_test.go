package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dinapappor/go-dashcam/config"
)

const otherFinishedRecording = "dashcam_20261123-083000.mp4"

type deleteSetup struct {
	handler http.Handler
	dir     string
	parent  string
}

func newDeleteSetup(t *testing.T) deleteSetup {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "videos")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	writeRecording(t, dir, otherFinishedRecording, 16)
	writeRecording(t, dir, finishedRecording, 16)
	writeRecording(t, dir, activeRecording, 16)
	writeRecording(t, dir, "notes.txt", 16)
	writeRecording(t, dir, "dashcam_20261123-080000.mkv", 16)
	writeRecording(t, parent, "dashcam_20261123-070000.mp4", 16)
	return deleteSetup{
		handler: newHandler(config.Config{OutputDirectory: dir}),
		dir:     dir,
		parent:  parent,
	}
}

func deleteRequest(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
	return recorder
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s should still exist: %v", filepath.Base(path), err)
	}
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s should be deleted, Stat error = %v", filepath.Base(path), err)
	}
}

func assertNothingDeleted(t *testing.T, setup deleteSetup) {
	t.Helper()
	for _, name := range []string{
		otherFinishedRecording,
		finishedRecording,
		activeRecording,
		"notes.txt",
		"dashcam_20261123-080000.mkv",
	} {
		assertExists(t, filepath.Join(setup.dir, name))
	}
	assertExists(t, filepath.Join(setup.parent, "dashcam_20261123-070000.mp4"))
}

func TestDeleteRecording(t *testing.T) {
	setup := newDeleteSetup(t)

	recorder := deleteRequest(t, setup.handler, "/video/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	assertGone(t, filepath.Join(setup.dir, finishedRecording))
	assertExists(t, filepath.Join(setup.dir, otherFinishedRecording))
	assertExists(t, filepath.Join(setup.dir, activeRecording))
	assertExists(t, filepath.Join(setup.dir, "notes.txt"))
	assertExists(t, filepath.Join(setup.dir, "dashcam_20261123-080000.mkv"))
	assertExists(t, filepath.Join(setup.parent, "dashcam_20261123-070000.mp4"))
}

func TestDeleteRepliesWithUpdatedList(t *testing.T) {
	setup := newDeleteSetup(t)

	recorder := deleteRequest(t, setup.handler, "/video/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
	body := recorder.Body.String()
	if strings.Contains(body, finishedRecording) {
		t.Errorf("reply still lists the deleted recording:\n%s", body)
	}
	for _, want := range []string{otherFinishedRecording, activeRecording, "Recording now"} {
		if !strings.Contains(body, want) {
			t.Errorf("reply does not contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<html") {
		t.Errorf("reply is a fragment and must not contain <html:\n%s", body)
	}
}

func TestDeleteRejectsBadNames(t *testing.T) {
	for _, tt := range badRecordingNames {
		t.Run(tt.name, func(t *testing.T) {
			setup := newDeleteSetup(t)

			recorder := deleteRequest(t, setup.handler, "/video/"+tt.path)

			if recorder.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
			assertNothingDeleted(t, setup)
		})
	}
}

func TestDeleteActiveRecording(t *testing.T) {
	setup := newDeleteSetup(t)

	recorder := deleteRequest(t, setup.handler, "/video/"+activeRecording)

	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	assertNothingDeleted(t, setup)
}

func TestDeleteRemoveFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can delete from a read-only directory")
	}
	setup := newDeleteSetup(t)
	if err := os.Chmod(setup.dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(setup.dir, 0700) })

	recorder := deleteRequest(t, setup.handler, "/video/"+finishedRecording)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertExists(t, filepath.Join(setup.dir, finishedRecording))
}

func attributeValue(body, element, attribute string) (string, bool) {
	start := strings.Index(body, element)
	if start < 0 {
		return "", false
	}
	end := strings.Index(body[start:], ">")
	if end < 0 {
		return "", false
	}
	tag := body[start : start+end]
	_, after, found := strings.Cut(tag, attribute+`="`)
	if !found {
		return "", false
	}
	value, _, found := strings.Cut(after, `"`)
	return value, found
}

func TestRecordingsPageDeleteButtons(t *testing.T) {
	setup := newDeleteSetup(t)

	recorder := get(t, setup.handler, "/recordings")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()

	deleteAttribute := `hx-delete="/video/` + finishedRecording + `"`
	if !strings.Contains(body, deleteAttribute) {
		t.Fatalf("recordings page does not contain %q:\n%s", deleteAttribute, body)
	}
	start := strings.LastIndex(body[:strings.Index(body, deleteAttribute)], "<")
	button := body[start:]

	confirm, found := attributeValue(button, "<", "hx-confirm")
	if !found {
		t.Errorf("delete element has no hx-confirm:\n%s", body)
	} else if !strings.Contains(confirm, finishedRecording) {
		t.Errorf("hx-confirm = %q, want it to contain %q", confirm, finishedRecording)
	}
	if _, found := attributeValue(button, "<", "hx-target"); !found {
		t.Errorf("delete element has no hx-target:\n%s", body)
	}

	if strings.Contains(body, `hx-delete="/video/`+activeRecording+`"`) {
		t.Errorf("active recording has a delete button:\n%s", body)
	}
}
