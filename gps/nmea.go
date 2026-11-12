package gps

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Sentence interface {
	Kind() string
}

type RMC struct {
	Time      time.Time // UTC
	Valid     bool
	Latitude  float64
	Longitude float64
	SpeedKts  float64
	Bearing   float64
}

type GGA struct {
	FixQuality int // 0 = no fix
	Satellites int
	HDOP       float64
	Altitude   float64
}

func (RMC) Kind() string {
	return "RMC"
}

func (GGA) Kind() string {
	return "GGA"
}

// Add methods for knots to kmh/mph
func (r RMC) KMH() float64 {
	return r.SpeedKts * 1.852
}

func (r RMC) MPH() float64 {
	return r.SpeedKts * 1.15078
}

func calculateChecksum(nmeaSentence string) uint8 {
	var checksum uint8

	for i := 0; i < len(nmeaSentence); i++ {
		checksum = checksum ^ nmeaSentence[i]
	}
	return checksum
}

func validateSentence(nmeaSentence string) (string, error) {
	sentence, found := strings.CutPrefix(nmeaSentence, "$")

	if !found {
		// If it doesn't start with $
		// we reject it
		return "", fmt.Errorf("A proper NMEA sentence needs to start with $")
	}

	// get the checksum and what's before *
	// checksumHex is to denote it's the original checksum
	sentence, checksumHex, found := strings.Cut(sentence, "*")

	if !found {
		// This nmeaSentence most likely is not a real NMEA
		// sentence. We reject it
		return "", fmt.Errorf("Missing checksum separator (*)")
	}

	receivedChecksum, err := strconv.ParseUint(checksumHex, 16, 8)

	if err != nil {
		return "", fmt.Errorf("Could not parse received checksum into an uint8: %w", err)
	}

	// Calculate checksum for later comparison with
	// original sentence checksum
	calculatedChecksum := calculateChecksum(sentence)

	if calculatedChecksum != uint8(receivedChecksum) {
		return "", fmt.Errorf("Calculated checksum did not match got: %02X, expected: %02X", receivedChecksum, calculatedChecksum)
	}

	return sentence, nil
}

func parseCoordinate(value string, hemisphere string) (float64, error) {

	// default value, Southern and western hemisphere has negative values
	negative := false

	switch hemisphere {
	case "N", "E":
		// Do nothing
	case "S", "W":
		negative = true
	default:
		return 0, fmt.Errorf("There is no such hemisphere as %q", hemisphere)
	}
	parsedValue, err := strconv.ParseFloat(value, 64)

	if err != nil {
		return 0, fmt.Errorf("Could not parse float: %w", err)
	}
	degrees := math.Floor(parsedValue / 100)
	minutes := parsedValue - (degrees * 100)

	result := degrees + (minutes / 60)

	// This triggers for southern and western hemispheres
	if negative {
		return -result, nil
	}
	return result, nil
}

func ParseNMEA(line string) (Sentence, error) {
	body, err := validateSentence(line)
	if err != nil {
		return nil, err
	}

	fields := strings.Split(body, ",")
	if len(fields[0]) != 5 {
		return nil, fmt.Errorf("bad sentence name %q", fields[0])
	}
	kind := fields[0][2:] // "GPRMC" -> "RMC", "GNRMC" -> "RMC"

	switch kind {
	case "RMC":
		return parseRMC(fields[1:])
	case "GGA":
		return parseGGA(fields[1:])
	default:
		return nil, fmt.Errorf("unsupported sentence %q", kind)
	}
}

func parseRMC(fields []string) (RMC, error) {

	rmc := RMC{}

	if len(fields) < 11 {
		return RMC{}, fmt.Errorf("Not enough fields, got %d", len(fields))
	}

	switch fields[1] {
	case "A":
		rmc.Valid = true
	case "V":
		// no fix - V is still valid, so we return nil for error
		return RMC{Valid: false}, nil
	default:
		// This unknown value, return error
		return RMC{}, fmt.Errorf("Unknown RMC status %q", fields[1])
	}

	latitude, err := parseCoordinate(fields[2], fields[3])
	if err != nil {
		return RMC{}, fmt.Errorf("Could not parse latitude: %w", err)
	}

	rmc.Latitude = latitude

	longitude, err := parseCoordinate(fields[4], fields[5])

	if err != nil {
		return RMC{}, fmt.Errorf("Could not parse longitude: %w", err)
	}

	rmc.Longitude = longitude

	speed, err := strconv.ParseFloat(fields[6], 64)

	if err != nil {
		return RMC{}, fmt.Errorf("Could not parse speed: %w", err)
	}

	rmc.SpeedKts = speed

	bearing, err := strconv.ParseFloat(fields[7], 64)

	if err != nil {
		// This is optional, and receivers leave it empty sometimes when you're still
		if fields[7] == "" {
			rmc.Bearing = 0
		} else {
			return RMC{}, fmt.Errorf("Unable to parse bearing:%s, %w", fields[7], err)
		}
	}
	rmc.Bearing = bearing

	timestamp, err := time.Parse("020106150405", fields[8]+fields[0])
	if err != nil {
		return RMC{}, fmt.Errorf("Could not parse time %q: %w", fields[8]+fields[0], err)
	}

	rmc.Time = timestamp

	return rmc, nil
}

func parseGGA(fields []string) (GGA, error) {
	return GGA{}, fmt.Errorf("not implemented")
}
