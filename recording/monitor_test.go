package recording

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonitorIntervalZero(t *testing.T) {
	ctx := context.Background()

	err := Monitor(ctx, t.TempDir(), 0)

	if err == nil {
		t.Fatalf("We should get an error from a zero interval")
	}
}

func TestMonitorIntervalNegative(t *testing.T) {
	ctx := context.Background()

	err := Monitor(ctx, t.TempDir(), -999)

	if err == nil {
		t.Fatalf("We should get an error from a negative interval")
	}
}

func TestMonitorIntervalNonExistingDir(t *testing.T) {
	ctx := context.Background()

	err := Monitor(ctx, filepath.Join(t.TempDir(), "fake"), 5*time.Second)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("We should get an error from a non-existing directory")
	}
}

func TestMonitorCancellation(t *testing.T) {
	freeSpace = func(path string) (uint64, error) {
		return MinSpace, nil
	}
	t.Cleanup(func() { freeSpace = FreeSpace })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Monitor(ctx, t.TempDir(), time.Hour)

	if err != nil {
		t.Fatalf("We should get a nil when context is cancelled")
	}
}

func TestMonitorCancellationWhileRunning(t *testing.T) {
	freeSpace = func(path string) (uint64, error) {
		return MinSpace, nil
	}
	t.Cleanup(func() { freeSpace = FreeSpace })

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A buffered channel lets the goroutine finish even if the test times out.
	result := make(chan error, 1)
	go func() {
		result <- Monitor(ctx, dir, time.Hour)
	}()

	// The monitor should keep running until we cancel it.
	select {
	case err := <-result:
		t.Fatalf("Monitor returned before cancellation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	cancel()

	// Cancellation should wake the monitor without waiting for its hourly tick.
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Monitor returned an error after cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Monitor did not stop within one second after cancellation")
	}
}
