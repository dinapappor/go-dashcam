package recording

import (
	"fmt"
	"time"
	"context"
)

// Should be configurable in the future
const MinSpace = 2 << 30 // 2 GiB

func Monitor(ctx context.Context, path string, interval time.Duration) error {

	if interval <= 0 {
		return fmt.Errorf("We need a positive otherwise time.Newticker will panic")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run loop where we check free space
	// and also make sure handle context and stop
	// the loop if we're shutting down
	for {
		// call cleanup, we have minspace check in there

		err := Cleanup(path)

		if err != nil {
			return fmt.Errorf("Could not run cleanup: %w", err)
		}
		select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return nil
		}
	}
}
