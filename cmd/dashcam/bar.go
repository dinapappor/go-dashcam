package main

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/dinapappor/go-dashcam/gps"
)

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
