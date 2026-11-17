package clock

import (
	"os"
)

// basically for when we do have NTP time and still want to to display time
// despite having no GPS fix

const TimesyncdMarker = "/run/systemd/timesync/synchronized"

func Synced(path string) bool {

	_, err := os.Stat(path)

	if err != nil {
		return false
	}
	return true
}
