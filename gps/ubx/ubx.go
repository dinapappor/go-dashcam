package ubx

import (
	"encoding/binary"
	"fmt"
	"io"
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

// message classes

const (
	classCFG  = 0x06
	classNMEA = 0xF0
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
