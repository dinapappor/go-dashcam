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

// if we lose fix we make sure we show NO GPS FIX no later than 3 seconds after last good fix
const maxFixAge = 3 * time.Second

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

func signalBars(fix gps.Fix) string {
	if !fix.Trusted() {
		return "    "
	}
	switch {
	case fix.GGA.HDOP < 1:
		return "▁▃▅▇"
	case fix.GGA.HDOP < 2:
		return "▁▃▅ "
	case fix.GGA.HDOP < 5:
		return "▁▃  "
	default:
		return "    "
	}
}

func micText(configured bool, present bool) string {
	if configured && present {
		return "MIC on"
	}

	if configured && !present {
		return "MIC off"
	}

	return ""
}

func barText(fix gps.Fix, unit string, now time.Time, clockSynced bool, mic string) string {

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

	noFix := "No GPS fix"
	if mic != "" {
		noFix = fmt.Sprintf("%-24s  %s", noFix, mic)
	}

	if fix.Trusted() {
		lat := formatCoordinate(fix.RMC.Latitude, "N", "S", 7)
		lon := formatCoordinate(fix.RMC.Longitude, "E", "W", 8)
		timeText := fix.RMC.Time.Local().Format("2006-01-02 15:04:05")
		signalBar := signalBars(fix)
		line := fmt.Sprintf("%3.0f %s  %s  %s %s  %s", speed, speedUnit, timeText, lat, lon, signalBar)
		if mic != "" {
			line += "  " + mic
		}
		return line
	} else {
		if clockSynced {
			ntpDateTime := now.Local().Format("2006-01-02 15:04:05")
			return fmt.Sprintf("--- %s  %s  %s", speedUnit, ntpDateTime, noFix)
		}
	}
	return fmt.Sprintf("--- %s  ---------- --:--:--  %s", speedUnit, noFix)
}

func barLoop(ctx context.Context, bar *atomic.Pointer[barImage], barConfig overlay.Config, speedUnit string, mic string, fixes <-chan gps.Fix) {

	var lastText string
	var fix gps.Fix
	var lastFixTime time.Time
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// we're shutting down
			return
		case <-ticker.C:
		case fix = <-fixes:
			lastFixTime = barNow()
		}

		// if we havent had an update for 3 secs
		// we treat it as no fix
		if barNow().Sub(lastFixTime) > maxFixAge {
			fix = gps.Fix{}
		}

		text := barText(fix, speedUnit, barNow(), barClockSynced(), mic)

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
