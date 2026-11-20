package ubx

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// NMEA message ids
const (
	nmeaGGA = 0x00
	nmeaGLL = 0x01
	nmeaGSA = 0x02
	nmeaGSV = 0x03
	nmeaRMC = 0x04
	nmeaVTG = 0x05
	nmeaGST = 0x07
)

// classMGA message ids
const (
	MGAIni = 0x40
)

// message classes
const (
	classCFG  = 0x06
	classNMEA = 0xF0
	classMGA  = 0x13
)

// CFG message
const (
	cfgMSG  = 0x01
	cfgRATE = 0x08
	cfgNAV5 = 0x24
)

// dynmodel
const (
	dynModelPortable     = 0
	dynModelStationary   = 2
	dynModelPedestrian   = 3
	dynModelAutomotive   = 4
	dynModelAtSea        = 5
	dynModelAirborneLt1g = 6
	dynModelAirborneLt2g = 7
	dynModelAirborneLt4g = 8
)

// on/off
const (
	ON  = 0x01
	OFF = 0x00
)

const MgaIniTypeTimeUTC = 0x10

// This is basically like formatSentence for pmtk
func frame(class byte, id byte, payload []byte) []byte {
	// 0xB5 = µ, 0x62 = b, ( µ == ublox)
	msg := []byte{0xB5, 0x62}
	// class is message group (CG, MON, ACK, NAV etc)
	// id is RATE, NAV, GNSS etc
	msg = append(msg, class, id)
	// add payload length
	msg = binary.LittleEndian.AppendUint16(msg, uint16(len(payload)))
	// add actual payload
	msg = append(msg, payload...)

	// calculate checksums
	var ckA uint8
	var ckB uint8

	for _, b := range msg[2:] {
		ckA += b
		ckB += ckA

	}

	// add checksums
	msg = append(msg, ckA, ckB)

	return msg
}

func Send(w io.Writer, class byte, id byte, payload []byte) error {
	pl := frame(class, id, payload)

	n, err := w.Write(pl)
	if err != nil {
		return fmt.Errorf("Could not send ubx frame: %w", err)
	}

	if n != len(pl) {
		return fmt.Errorf("Expected to send %d bytes, sent %d bytes", len(pl), n)
	}
	return nil
}

func Configure(w io.Writer) error {
	// We ignore these because we're
	// only interested in GGA (quality of fix etc)
	// and RMC (Recommended minimum data)
	// /
	unwantedSentences := []byte{nmeaGLL, nmeaGSA, nmeaGSV, nmeaVTG, nmeaGST}

	for _, us := range unwantedSentences {
		err := Send(w, classCFG, cfgMSG, []byte{classNMEA, us, OFF})

		if err != nil {
			return fmt.Errorf("Unable to shut off sentence id 0x%02X: %w", us, err)
		}
	}

	// Set update rate to 5hz (0xC8)
	err := Send(w, classCFG, cfgRATE, []byte{0xC8, 0x00, 0x01, 0x00, 0x01, 0x00})
	if err != nil {
		return fmt.Errorf("Unable to set update rate to 5hz: %w", err)
	}

	// Set navmode, hardcoded to automotive for now
	payload := make([]byte, 36)
	payload[0] = ON
	payload[2] = dynModelAutomotive

	err = Send(w, classCFG, cfgNAV5, payload)

	if err != nil {
		return fmt.Errorf("Unable to set navmode: %w", err)
	}
	return nil
}

// if we have NTP time this might be useful to acquire a
// fix faster
func InitTime(w io.Writer, t time.Time) error {

	// Hardcoding this because 2 was what I could
	// find about ntp
	var accuracySeconds uint16 = 2

	utc := t.UTC()
	payload := make([]byte, 24)

	payload[0] = MgaIniTypeTimeUTC
	payload[1] = 0x00
	payload[2] = 0x00
	payload[3] = 0x80
	binary.LittleEndian.PutUint16(payload[4:6], uint16(utc.Year()))
	payload[6] = byte(utc.Month())
	payload[7] = byte(utc.Day())
	payload[8] = byte(utc.Hour())
	payload[9] = byte(utc.Minute())
	payload[10] = byte(utc.Second())
	payload[11] = 0x00
	binary.LittleEndian.PutUint32(payload[12:16], uint32(utc.Nanosecond()))
	binary.LittleEndian.PutUint16(payload[16:18], accuracySeconds)

	err := Send(w, classMGA, MGAIni, payload)

	if err != nil {
		return fmt.Errorf("Could not send initial time to GPS: %w", err)
	}

	return nil
}
