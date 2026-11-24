package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/dinapappor/go-dashcam/recording"
)

type recordingInfo struct {
	Name   string
	Size   int64
	Time   time.Time
	Active bool
}

// for once I feel like hackerboi
func formatSize(bytes int64) string {

	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	byteUnits := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}

	size := float64(bytes)

	divisionCount := 0
	for size >= 1024 {
		size = size / 1024
		divisionCount++
		if divisionCount == len(byteUnits)-1 {
			break
		}
	}

	return fmt.Sprintf("%.1f %s", size, byteUnits[divisionCount])

}

func listRecordingInfo(dir string) ([]recordingInfo, error) {

	_, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("Unable to find directory %s: %w", dir, err)
	}

	var recordings []recordingInfo

	files, err := recording.ListRecordings(dir)

	if err != nil {
		return []recordingInfo{}, fmt.Errorf("Could not list recordings: %w", err)
	}

	slices.Reverse(files)

	for _, file := range files {
		recording := recordingInfo{}
		recording.Name = filepath.Base(file)
		f, err := os.Stat(file)
		if err != nil {
			return []recordingInfo{}, fmt.Errorf("Could not stat file: %w", err)
		}
		timestampFromFile := strings.TrimPrefix(filepath.Base(file), "dashcam_")
		timestampFromFile = strings.TrimSuffix(timestampFromFile, ".mp4")
		timestamp, err := time.ParseInLocation("20060102-150405", timestampFromFile, time.Local)
		if err != nil {
			log.Printf("Could not parse date: %s", err)
		} else {
			recording.Time = timestamp
		}
		recording.Size = f.Size()
		recording.Active = false
		recordings = append(recordings, recording)
	}

	// first one should be active

	if len(recordings) > 0 {
		recordings[0].Active = true
	}
	return recordings, nil
}
