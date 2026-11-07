package recording

import (
	"os"
	"fmt"
)

func Cleanup(path string) error {
	for {
		free, err := FreeSpace(path)
		if err != nil {
			return fmt.Errorf("Could not check free space: %w", err)
		}

		// We return early if there already is space
		if free >= MinSpace {
			return nil
		}

		files, err := ListRecordings(path)

		if err != nil {
			return fmt.Errorf("Unable to list recordings: %w", err)
		}

		if len(files) <2 {
			return fmt.Errorf("Only one file left, bailing")
		}

		err = os.Remove(files[0])

		if err != nil {
			return fmt.Errorf("Unable to remove oldest file %s: %w", files[0], err)
		}
	}
}
