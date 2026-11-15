package pmtk

import (
	"fmt"
	"io"

	"github.com/dinapappor/go-dashcam/gps"
)

func formatSentence(body string) string {
	checksum := gps.CalculateChecksum(body)

	return fmt.Sprintf("$%s*%02X\r\n", body, checksum)
}

func Send(writer io.Writer, body string) error {
	_, err := writer.Write([]byte(formatSentence(body)))

	if err != nil {
		return fmt.Errorf("Unable to send pmtk command %q: %w", body, err)
	}
	return nil
}

func Configure(w io.Writer) error {

	// Statically configure these for now
	// later maybe something for config

	// RMC+GGA
	err := Send(w, "PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0")

	if err != nil {
		return fmt.Errorf("Unable to set RMC+GGA: %w", err)
	}

	// 5 Hz updates
	err = Send(w, "PMTK220,200")

	if err != nil {
		return fmt.Errorf("Unable to set 5Hz updates: %w", err)
	}

	return nil
}
