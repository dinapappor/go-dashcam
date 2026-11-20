package pmtk

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestFormatSentence(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "10 Hz update rate",
			body: "PMTK220,100",
			want: "$PMTK220,100*2F\r\n",
		},
		{
			name: "1 Hz update rate",
			body: "PMTK220,1000",
			want: "$PMTK220,1000*1F\r\n",
		},
		{
			name: "115200 baud",
			body: "PMTK251,115200",
			want: "$PMTK251,115200*1F\r\n",
		},
		{
			name: "RMC and GGA only",
			body: "PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0",
			want: "$PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0*28\r\n",
		},
		{
			name: "acknowledgement sentence",
			body: "PMTK001,220,3",
			want: "$PMTK001,220,3*30\r\n",
		},
		{
			name: "small checksum is zero padded",
			body: "PQ",
			want: "$PQ*01\r\n",
		},
		{
			name: "empty body",
			body: "",
			want: "$*00\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSentence(tt.body)
			if got != tt.want {
				t.Errorf("formatSentence(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestSendWritesFormattedSentence(t *testing.T) {
	var buf bytes.Buffer

	err := Send(&buf, "PMTK220,100")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := "$PMTK220,100*2F\r\n"
	if got := buf.String(); got != want {
		t.Errorf("Send wrote %q, want %q", got, want)
	}
}

func TestSendTwoCommandsInOrder(t *testing.T) {
	var buf bytes.Buffer

	for _, body := range []string{"PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0", "PMTK220,100"} {
		if err := Send(&buf, body); err != nil {
			t.Fatalf("Send(%q): %v", body, err)
		}
	}

	want := "$PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0*28\r\n$PMTK220,100*2F\r\n"
	if got := buf.String(); got != want {
		t.Errorf("Send wrote %q, want %q", got, want)
	}
}

func TestSendWriteError(t *testing.T) {
	writeErr := errors.New("bus gone")

	err := Send(failingWriter{err: writeErr}, "PMTK220,100")
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

func TestConfigureSendsCommandsInOrder(t *testing.T) {
	var buf bytes.Buffer

	if err := Configure(&buf); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	want := "$PMTK251,115200*1F\r\n" +
		"$PMTK314,0,1,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0*28\r\n" +
		"$PMTK220,100*2F\r\n"
	if got := buf.String(); got != want {
		t.Errorf("Configure wrote %q, want %q", got, want)
	}
}

func TestConfigureWriteError(t *testing.T) {
	writeErr := errors.New("bus gone")

	err := Configure(failingWriter{err: writeErr})
	if !errors.Is(err, writeErr) {
		t.Fatalf("Configure error = %v, want wrapped %v", err, writeErr)
	}
}

func TestConfigureStopsAfterFirstError(t *testing.T) {
	w := &countingWriter{failOn: 1, err: errors.New("bus gone")}

	if err := Configure(w); err == nil {
		t.Fatal("Configure succeeded, want error")
	}
	if w.calls != 1 {
		t.Errorf("Configure made %d writes after the first failed, want 1 in total", w.calls)
	}
}

type countingWriter struct {
	calls  int
	failOn int
	err    error
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failOn {
		return 0, w.err
	}
	return len(p), nil
}

func TestInitTime(t *testing.T) {
	plus2 := time.FixedZone("+02:00", 2*60*60)
	minus5 := time.FixedZone("-05:00", -5*60*60)

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "UTC",
			time: time.Date(2026, 11, 20, 13, 45, 30, 0, time.UTC),
			want: "$PMTK740,2026,11,20,13,45,30*35\r\n",
		},
		{
			name: "same instant in +02:00",
			time: time.Date(2026, 11, 20, 15, 45, 30, 0, plus2),
			want: "$PMTK740,2026,11,20,13,45,30*35\r\n",
		},
		{
			name: "nanoseconds are dropped",
			time: time.Date(2026, 11, 20, 13, 45, 30, 999999999, time.UTC),
			want: "$PMTK740,2026,11,20,13,45,30*35\r\n",
		},
		{
			name: "single digits are zero padded",
			time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			want: "$PMTK740,2026,01,02,03,04,05*36\r\n",
		},
		{
			name: "local date differs from UTC date",
			time: time.Date(2025, 12, 31, 18, 59, 59, 0, minus5),
			want: "$PMTK740,2025,12,31,23,59,59*34\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			if err := InitTime(&buf, tt.time); err != nil {
				t.Fatalf("InitTime: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("InitTime wrote %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitTimeWriteError(t *testing.T) {
	writeErr := errors.New("device gone")

	err := InitTime(failingWriter{err: writeErr}, time.Date(2026, 11, 20, 13, 45, 30, 0, time.UTC))
	if !errors.Is(err, writeErr) {
		t.Fatalf("InitTime error = %v, want wrapped %v", err, writeErr)
	}
}
