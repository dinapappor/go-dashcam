package main

import (
	"fmt"
	"log"

	"github.com/dinapappor/go-dashcam/gps"
)

func main() {
	f, err := gps.OpenI2C("/dev/i2c-1", 0x10)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = gps.ReadSentences(f, func(s gps.Sentence) {
		fmt.Printf("%s %+v\n", s.Kind(), s)
	})
	if err != nil {
		log.Fatal(err)
	}
}
