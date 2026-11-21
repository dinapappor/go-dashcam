package video

import (
	"errors"
)

func AudioInputArgs(device string) ([]string, error) {

	if device == "" {
		return nil, errors.New("Jeesos karen, you must give me a device")
	}

	inputArgs := []string{"-thread_queue_size", "512", "-f", "alsa", "-i", device}

	return inputArgs, nil
}

func AudioOutputArgs() []string {
	return []string{"-copyts", "-start_at_zero", "-c:a", "aac", "-ac", "2"}
}

func VideoSyncArgs() []string {
	return []string{"-use_wallclock_as_timestamps", "1", "-probesize", "32", "-analyzeduration", "0", "-fflags", "nobuffer"}
}
