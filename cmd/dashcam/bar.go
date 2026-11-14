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

func barText(fix gps.Fix) string {
	if !fix.RMC.Valid {
		return "No GPS fix"
	}
	return fmt.Sprintf("%.0f km/h %s", fix.RMC.KMH(), fix.RMC.Time.Local().Format("2006-01-02 15:04:05"))
}
