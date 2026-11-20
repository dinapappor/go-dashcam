package pmtk

import (
	"fmt"
	"io"
	"time"

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

	// baudrate
	err := Send(w, "PMTK251,115200")

	if err != nil {
		return fmt.Errorf("Unable to set baudrate: %w", err)
	}

	// RMC+GGA
	err = Send(w, "PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0")

	if err != nil {
		return fmt.Errorf("Unable to set RMC+GGA: %w", err)
	}

	// 10 Hz updates
	err = Send(w, "PMTK220,100")

	if err != nil {
		return fmt.Errorf("Unable to set 10Hz updates: %w", err)
	}

	return nil
}

func InitTime(w io.Writer, t time.Time) error {
	// I am a genious here
	timeString := t.UTC().Format("2006,01,02,15,04,05")
	err := Send(w, fmt.Sprintf("PMTK740,%s", timeString))

	if err != nil {
		return fmt.Errorf("Could not send initial time to pmtk device: %w", err)
	}
	return nil

}
