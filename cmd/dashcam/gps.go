package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dinapappor/go-dashcam/clock"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/gps"
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
		f, err := gps.OpenSerial(cfg.Device)

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

func readGPS(ctx context.Context, r io.ReadCloser, fixes chan<- gps.Fix) error {

	defer r.Close()

	stop := context.AfterFunc(ctx, func() {
		r.Close()
	})
	defer stop()

	var fix gps.Fix

	err := gps.ReadSentences(r, func(s gps.Sentence) {
		fix.Update(s)
		select {
		case fixes <- fix:
		case <-ctx.Done():
		}

	})

	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("Read GPS: %w", err)
	}

	return nil
}
