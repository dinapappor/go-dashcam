package main

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/dinapappor/go-dashcam/gps"
	"github.com/dinapappor/go-dashcam/overlay"
)

func readGPS(path string, address int, bar *atomic.Pointer[barImage], barConfig overlay.Config) error {

	f, err := gps.OpenI2C(path, address)

	if err != nil {
		return fmt.Errorf("Could not read gps: %w", err)
	}

	defer f.Close()

	var fix gps.Fix
	var lastText string

	err = gps.ReadSentences(f, func(s gps.Sentence) {
		fix.Update(s)
		text := barText(fix)

		// Don't update if there are no updates
		if text == lastText {
			return
		}

		barConfig.Text = text
		g, a, err := overlay.RenderOverlay(barConfig)

		if err != nil {
			log.Printf("Could not render bottom bar: %v", err)
			return
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

	})

	if err != nil {
		return fmt.Errorf("Read GPS: %w", err)
	}

	return nil
}
