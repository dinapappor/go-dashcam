package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/dinapappor/go-dashcam/gps"
	"github.com/dinapappor/go-dashcam/overlay"
)

// maybe do this configurable in the future
const minSpeedKts = 1.08 // 2 km/h or 1.23 mph because of drift

type barImage struct {
	gray  *image.Gray
	alpha *image.Alpha
}

func formatCoordinate(value float64, positive, negative string, width int) string {
	letter := positive

	if value < 0 {
		letter = negative
	}

	return fmt.Sprintf("%s%0*.4f", letter, width, math.Abs(value))
}

func barText(fix gps.Fix, unit string, now time.Time, clockSynced bool) string {

	var speedUnit string
	var speed float64

	if unit == "mph" {
		speedUnit = "mph"
		speed = fix.RMC.MPH()
	} else {
		speedUnit = "km/h"
		speed = fix.RMC.KMH()
	}

	// to not get speeds while standing still
	if fix.RMC.SpeedKts < minSpeedKts {
		speed = 0
	}

	if fix.RMC.Valid {
		lat := formatCoordinate(fix.RMC.Latitude, "N", "S", 7)
		lon := formatCoordinate(fix.RMC.Longitude, "E", "W", 8)
		timeText := fix.RMC.Time.Local().Format("2006-01-02 15:04:05")
		return fmt.Sprintf("%3.0f %s  %s  %s %s", speed, speedUnit, timeText, lat, lon)
	} else {
		if clockSynced {
			ntpDateTime := now.Local().Format("2006-01-02 15:04:05")
			return fmt.Sprintf("--- %s  %s  No GPS fix", speedUnit, ntpDateTime)
		}
	}
	return fmt.Sprintf("--- %s  ---------- --:--:--  No GPS fix", speedUnit)
}

func barLoop(ctx context.Context, bar *atomic.Pointer[barImage], barConfig overlay.Config, speedUnit string, fixes <-chan gps.Fix) {

	var lastText string
	var fix gps.Fix
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// we're shutting down
			return
		case <-ticker.C:
		case fix = <-fixes:
		}

		text := barText(fix, speedUnit, barNow(), barClockSynced())

		if text == lastText {
			continue
		}
		barConfig.Text = text
		g, a, err := overlay.RenderOverlay(barConfig)
		if err != nil {
			log.Printf("Could not render bottom bar: %v", err)
			continue
		}

		// Now write to the atomic pointer image
		bar.Store(
			&barImage{
				gray:  g,
				alpha: a,
			},
		)

		// Keep for next comparison
		lastText = text
	}

}
