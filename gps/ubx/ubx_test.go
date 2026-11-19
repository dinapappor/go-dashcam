package ubx

import (
	"bytes"
	"errors"
	"testing"
)

func TestFrame(t *testing.T) {
	tests := []struct {
		name    string
		class   byte
		id      byte
		payload []byte
		want    []byte
	}{
		// measRate 100 ms (10 Hz), navRate 1, timeRef GPS
		{
			name:    "CFG-RATE 10 Hz",
			class:   0x06,
			id:      0x08,
			payload: []byte{0x64, 0x00, 0x01, 0x00, 0x01, 0x00},
			want:    []byte{0xB5, 0x62, 0x06, 0x08, 0x06, 0x00, 0x64, 0x00, 0x01, 0x00, 0x01, 0x00, 0x7A, 0x12},
		},
		// measRate 1000 ms (1 Hz), navRate 1, timeRef GPS
		{
			name:    "CFG-RATE 1 Hz",
			class:   0x06,
			id:      0x08,
			payload: []byte{0xE8, 0x03, 0x01, 0x00, 0x01, 0x00},
			want:    []byte{0xB5, 0x62, 0x06, 0x08, 0x06, 0x00, 0xE8, 0x03, 0x01, 0x00, 0x01, 0x00, 0x01, 0x39},
		},
		// NMEA GSV (F0 03) at rate 0 = off
		{
			name:    "CFG-MSG GSV off",
			class:   0x06,
			id:      0x01,
			payload: []byte{0xF0, 0x03, 0x00},
			want:    []byte{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x03, 0x00, 0xFD, 0x15},
		},
		// empty payload = poll for version
		{
			name:    "MON-VER poll nil payload",
			class:   0x0A,
			id:      0x04,
			payload: nil,
			want:    []byte{0xB5, 0x62, 0x0A, 0x04, 0x00, 0x00, 0x0E, 0x34},
		},
		// empty payload = poll for version
		{
			name:    "MON-VER poll empty payload",
			class:   0x0A,
			id:      0x04,
			payload: []byte{},
			want:    []byte{0xB5, 0x62, 0x0A, 0x04, 0x00, 0x00, 0x0E, 0x34},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frame(tt.class, tt.id, tt.payload)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got % X, want % X", got, tt.want)
			}
		})
	}
}

func TestFrameLongPayloadLength(t *testing.T) {
	payload := make([]byte, 300)

	got := frame(0x06, 0x3E, payload)

	if len(got) != 2+4+300+2 {
		t.Fatalf("got %d bytes, want %d", len(got), 2+4+300+2)
	}
	if got[4] != 0x2C || got[5] != 0x01 {
		t.Errorf("got length bytes % X, want 2C 01", got[4:6])
	}
}

func TestFrameKeepsPayload(t *testing.T) {
	payload := []byte{0x64, 0x00, 0x01, 0x00, 0x01, 0x00}
	original := bytes.Clone(payload)

	frame(0x06, 0x08, payload)

	if !bytes.Equal(payload, original) {
		t.Errorf("payload changed to % X, want % X", payload, original)
	}
}

func TestSendWritesFrame(t *testing.T) {
	var buf bytes.Buffer

	// measRate 100 ms (10 Hz), navRate 1, timeRef GPS
	err := Send(&buf, 0x06, 0x08, []byte{0x64, 0x00, 0x01, 0x00, 0x01, 0x00})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := []byte{0xB5, 0x62, 0x06, 0x08, 0x06, 0x00, 0x64, 0x00, 0x01, 0x00, 0x01, 0x00, 0x7A, 0x12}
	if got := buf.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("Send wrote % X, want % X", got, want)
	}
}

func TestSendTwoMessagesInOrder(t *testing.T) {
	var buf bytes.Buffer

	// NMEA GSV (F0 03) at rate 0 = off
	if err := Send(&buf, 0x06, 0x01, []byte{0xF0, 0x03, 0x00}); err != nil {
		t.Fatalf("Send CFG-MSG: %v", err)
	}
	// empty payload = poll for version
	if err := Send(&buf, 0x0A, 0x04, nil); err != nil {
		t.Fatalf("Send MON-VER: %v", err)
	}

	want := []byte{
		0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x03, 0x00, 0xFD, 0x15,
		0xB5, 0x62, 0x0A, 0x04, 0x00, 0x00, 0x0E, 0x34,
	}
	if got := buf.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("Send wrote % X, want % X", got, want)
	}
}

func TestSendWriteError(t *testing.T) {
	writeErr := errors.New("device gone")

	err := Send(failingWriter{err: writeErr}, 0x0A, 0x04, nil)
	if !errors.Is(err, writeErr) {
		t.Fatalf("Send error = %v, want wrapped %v", err, writeErr)
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestConfigureWritesMessagesInOrder(t *testing.T) {
	var buf bytes.Buffer

	if err := Configure(&buf); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	want := [][]byte{
		// NMEA GLL (F0 01) at rate 0 = off
		{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x01, 0x00, 0xFB, 0x11},
		// NMEA GSA (F0 02) at rate 0 = off
		{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x02, 0x00, 0xFC, 0x13},
		// NMEA GSV (F0 03) at rate 0 = off
		{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x03, 0x00, 0xFD, 0x15},
		// NMEA VTG (F0 05) at rate 0 = off
		{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x05, 0x00, 0xFF, 0x19},
		// NMEA GST (F0 07) at rate 0 = off
		{0xB5, 0x62, 0x06, 0x01, 0x03, 0x00, 0xF0, 0x07, 0x00, 0x01, 0x1D},
		// measRate 200 ms (5 Hz), navRate 1, timeRef GPS
		{0xB5, 0x62, 0x06, 0x08, 0x06, 0x00, 0xC8, 0x00, 0x01, 0x00, 0x01, 0x00, 0xDE, 0x6A},
		// CFG-NAV5 mask 0x0001 (dynModel only), dynModel 4 = Automotive, rest zero
		append(append([]byte{0xB5, 0x62, 0x06, 0x24, 0x24, 0x00, 0x01, 0x00, 0x04}, make([]byte, 33)...), 0x53, 0x70),
	}

	got := buf.Bytes()
	for i, msg := range want {
		if !bytes.HasPrefix(got, msg) {
			t.Fatalf("message %d: got % X, want % X", i, got, msg)
		}
		got = got[len(msg):]
	}
	if len(got) != 0 {
		t.Errorf("unexpected trailing bytes % X", got)
	}
}

func TestConfigureWriteError(t *testing.T) {
	tests := []struct {
		name   string
		failAt int
	}{
		{"first CFG-MSG", 1},
		{"last CFG-MSG", 5},
		{"CFG-RATE", 6},
		{"CFG-NAV5", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeErr := errors.New("device gone")
			w := &failAtWriter{failAt: tt.failAt, err: writeErr}

			err := Configure(w)
			if !errors.Is(err, writeErr) {
				t.Fatalf("Configure error = %v, want wrapped %v", err, writeErr)
			}
			if w.writes != tt.failAt {
				t.Errorf("Configure made %d writes, want %d (stop at first error)", w.writes, tt.failAt)
			}
		})
	}
}

type failAtWriter struct {
	failAt int
	writes int
	err    error
}

func (w *failAtWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, w.err
	}
	return len(p), nil
}
