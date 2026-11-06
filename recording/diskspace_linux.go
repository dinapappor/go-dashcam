package recording

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Monitor and manage storage so we can remove older recordings

func FreeSpace(path string) (uint64, error) {
	fs := syscall.Statfs_t{}

	err := syscall.Statfs(path, &fs)

	if err != nil {
		return 0, fmt.Errorf("Unable to check disk space: %w", err)
	}
	return fs.Bavail * uint64(fs.Bsize), nil
}

func ListRecordings(path string) ([]string, error) {
	var foundRecordings []string

	files, err := os.ReadDir(path)

	if err != nil {
		return nil, fmt.Errorf("Could not list directory contents in %s: %w", path, err)
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "dashcam_") {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".mp4") {
			continue
		}
		if !file.Type().IsRegular() {
			// not a file
			continue
		}
		foundRecordings = append(foundRecordings, filepath.Join(path, file.Name()))
	}
	return foundRecordings, nil
}
