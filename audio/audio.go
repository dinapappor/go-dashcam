package audio

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Present(device string, procAsound string, devSnd string) bool {

	hw, found := strings.CutPrefix(device, "hw:")

	if !found {
		hw, found = strings.CutPrefix(device, "plughw:")
		if !found {
			// if we didn't found any of them
			// we get ffmpeg to report the error
			// yolo
			return true
		}
	}

	card, dev, found := strings.Cut(hw, ",")

	if !found {
		dev = "0"
	}

	// Now remove any card and dev to normalize

	card = strings.TrimPrefix(card, "CARD=")
	dev = strings.TrimPrefix(dev, "DEV=")

	_, err := strconv.Atoi(card)

	if err != nil {
		// this is name of a device (ie, roevmakkafon, or C93 and not from hv:1,0 etc)
		// so we check /proc/asound
		card, err = os.Readlink(fmt.Sprintf("%s/%s", procAsound, card))

		if err != nil {
			// no symlink found, no point in continuing
			return false
		}

		// remove "card" so we can use it in os.Stat() later
		card = strings.TrimPrefix(card, "card")

	}

	// now we check for actual device
	_, err = os.Stat(fmt.Sprintf("%s/pcmC%sD%sc", devSnd, card, dev))

	if err != nil {
		return false
	}

	return true
}
