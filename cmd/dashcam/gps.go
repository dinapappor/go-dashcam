package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/dinapappor/go-dashcam/gps"
	"github.com/dinapappor/go-dashcam/overlay"
)

func readGPS(ctx context.Context, path string, address int, bar *atomic.Pointer[barImage], barConfig overlay.Config, speedunit string) error {

	f, err := gps.OpenI2C(path, address)

	if err != nil {
		return fmt.Errorf("Could not read gps: %w", err)
	}

	defer f.Close()

	stop := context.AfterFunc(ctx, func() {
		f.Close()
	})
	defer stop()

	var fix gps.Fix
	var lastText string

	err = gps.ReadSentences(f, func(s gps.Sentence) {
		fix.Update(s)
		text := barText(fix, speedunit)

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
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("Read GPS: %w", err)
	}

	return nil
}
