package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHandlerRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantHTML   bool
	}{
		{name: "front page", method: http.MethodGet, path: "/", wantStatus: http.StatusOK, wantHTML: true},
		{name: "recordings page", method: http.MethodGet, path: "/recordings", wantStatus: http.StatusOK, wantHTML: true},
		{name: "post to recordings page", method: http.MethodPost, path: "/recordings", wantStatus: http.StatusMethodNotAllowed},
		{name: "post to stats", method: http.MethodPost, path: "/stats", wantStatus: http.StatusMethodNotAllowed},
		{name: "unknown path", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound},
		{name: "path below front page", method: http.MethodGet, path: "/index.html", wantStatus: http.StatusNotFound},
		{name: "post to front page", method: http.MethodPost, path: "/", wantStatus: http.StatusMethodNotAllowed},
		{name: "delete on front page", method: http.MethodDelete, path: "/", wantStatus: http.StatusMethodNotAllowed},
	}

	handler := newHandler(testConfig(t.TempDir()))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := send(t, handler, tt.method, tt.path, login(t, handler), nil)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if !tt.wantHTML {
				return
			}
			contentType := recorder.Header().Get("Content-Type")
			if !strings.HasPrefix(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
			if !strings.Contains(recorder.Body.String(), "<html") {
				t.Errorf("body has no <html tag: %q", recorder.Body.String())
			}
		})
	}
}

func TestServeAnswersAndStopsOnCancel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := "http://" + listener.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, listener, newHandler(testConfig(t.TempDir())))
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(address + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("serve returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after cancel")
	}

	if _, err := client.Get(address + "/login"); err == nil {
		t.Error("server still answers after serve returned")
	}
}

func TestServeAlreadyCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, listener, newHandler(testConfig(t.TempDir())))
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("serve returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return with a cancelled context")
	}
}

func TestServeClosedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, listener, newHandler(testConfig(t.TempDir())))
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("serve returned nil, want an error")
		}
		if errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve returned %v, want the listener error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return with a closed listener")
	}
}
