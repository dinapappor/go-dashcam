package gps

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
)

func TestPaddingFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		oneByte bool
	}{
		{name: "padding mid sentence", input: "$GN\nRMC*00\r\n", want: "$GNRMC*00\r\n"},
		{name: "padding before and after", input: "\n\n$GPGGA*00\r\n\n", want: "$GPGGA*00\r\n"},
		{name: "two sentences", input: "$GPRMC*00\r\n\n\n$GPGGA*00\r\n", want: "$GPRMC*00\r\n$GPGGA*00\r\n"},
		{name: "only padding", input: "\n\n\n", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "split across reads", input: "$GN\nRMC*00\r\n\n$GPGGA*00\r\n", want: "$GNRMC*00\r\n$GPGGA*00\r\n", oneByte: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r io.Reader = strings.NewReader(tt.input)
			if tt.oneByte {
				r = iotest.OneByteReader(r)
			}
			f := &paddingFilter{r: r}

			got, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPaddingFilterReader(t *testing.T) {
	f := &paddingFilter{r: strings.NewReader("$GN\nRMC*00\r\n\n$GPGGA*00\r\n")}
	if err := iotest.TestReader(f, []byte("$GNRMC*00\r\n$GPGGA*00\r\n")); err != nil {
		t.Error(err)
	}
}

func TestReadSentences(t *testing.T) {
	input := "\n\n$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A\r\n" +
		"\n$GPGSV,1,1,00*79\r\n" +
		"$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6B\r\n" +
		"$GP\nGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47\r\n\n"

	var kinds []string
	handle := func(s Sentence) {
		kinds = append(kinds, s.Kind())
	}

	err := ReadSentences(strings.NewReader(input), handle)
	if err != nil {
		t.Fatalf("ReadSentences returned error: %v", err)
	}

	want := []string{"RMC", "GGA"}
	if !slices.Equal(kinds, want) {
		t.Errorf("got kinds %v, want %v", kinds, want)
	}
}

func TestReadSentencesReaderError(t *testing.T) {
	readErr := errors.New("device gone")
	called := false

	err := ReadSentences(iotest.ErrReader(readErr), func(Sentence) { called = true })
	if !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
	if called {
		t.Error("handle was called without any input")
	}
}
