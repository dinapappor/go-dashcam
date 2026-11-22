package video

import (
	"errors"
	"strconv"
	"time"
)

func AudioInputArgs(device string, offsetValue time.Duration) ([]string, error) {

	if device == "" {
		return nil, errors.New("Jeesos karen, you must give me a device")
	}

	var inputArgs []string
	if offsetValue == 0 {
		inputArgs = []string{"-f", "alsa", "-i", device}
	} else {
		offset := strconv.FormatFloat(offsetValue.Seconds(), 'f', -1, 64)
		inputArgs = []string{"-itsoffset", offset, "-f", "alsa", "-i", device}

	}

	return inputArgs, nil
}

func AudioOutputArgs() []string {
	return []string{"-copyts", "-start_at_zero", "-c:a", "aac", "-ac", "2"}
}

func VideoSyncArgs() []string {
	return []string{"-use_wallclock_as_timestamps", "1", "-probesize", "32", "-analyzeduration", "0", "-fflags", "nobuffer"}
}
