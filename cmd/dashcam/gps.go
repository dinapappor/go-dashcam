package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dinapappor/go-dashcam/clock"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/gps"
	"github.com/dinapappor/go-dashcam/overlay"
)

func openGPS(cfg config.GPSConfig) (io.ReadWriteCloser, error) {
	//Decide on how to talk to the dewice
	switch cfg.Type {
	case "i2c":
		f, err := gps.OpenI2C(cfg.Device, cfg.AddressValue)

		if err != nil {
			return nil, fmt.Errorf("Unable to open %s: %w", cfg.Device, err)
		}
		return f, nil
	case "serial":
		f, err := os.OpenFile(cfg.Device, os.O_RDWR|syscall.O_NOCTTY, 0)

		if err != nil {
			return nil, fmt.Errorf("Unable to open %s: %w", cfg.Device, err)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("Unknown gps device type %q", cfg.Type)
	}
}

var barNow = time.Now

var barClockSynced = func() bool {
	return clock.Synced(clock.TimesyncdMarker)
}

func readGPS(ctx context.Context, r io.ReadCloser, bar *atomic.Pointer[barImage], barConfig overlay.Config, speedunit string) error {

	defer r.Close()

	stop := context.AfterFunc(ctx, func() {
		r.Close()
	})
	defer stop()

	var fix gps.Fix
	var lastText string

	err := gps.ReadSentences(r, func(s gps.Sentence) {
		fix.Update(s)
		text := barText(fix, speedunit, barNow(), barClockSynced())

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
