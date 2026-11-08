package main

import (
	"fmt"
	"os"
	"runtime/pprof"
)

// startCPUProfile returns cleanup that flushes the profile before closing its file.
func startCPUProfile(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("start CPU profile: %w", err)
	}
	return func() error {
		pprof.StopCPUProfile()
		return f.Close()
	}, nil
}
