package main

import (
	"fmt"
	"image"

	"github.com/dinapappor/go-dashcam/gps"
)

type barImage struct {
	gray  *image.Gray
	alpha *image.Alpha
}

func barText(fix gps.Fix, unit string) string {
	if !fix.RMC.Valid {
		return "No GPS fix"
	}

	var speedUnit string
	var speed float64

	if unit == "mph" {
		speedUnit = "mph"
		speed = fix.RMC.MPH()
	} else {
		speedUnit = "km/h"
		speed = fix.RMC.KMH()
	}

	return fmt.Sprintf("%.0f %s %s", speed, speedUnit, fix.RMC.Time.Local().Format("2006-01-02 15:04:05"))
}
