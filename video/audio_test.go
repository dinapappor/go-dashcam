package video

import (
	"slices"
	"testing"
	"time"
)

func TestAudioInputArgs(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		offset  time.Duration
		want    []string
		wantErr bool
	}{
		{
			name:   "hw device without offset",
			device: "hw:1",
			want:   []string{"-f", "alsa", "-i", "hw:1"},
		},
		{
			name:   "plughw with subdevice",
			device: "plughw:1,0",
			want:   []string{"-f", "alsa", "-i", "plughw:1,0"},
		},
		{
			name:   "default device",
			device: "default",
			want:   []string{"-f", "alsa", "-i", "default"},
		},
		{
			name:   "positive offset",
			device: "hw:1",
			offset: 100 * time.Millisecond,
			want:   []string{"-itsoffset", "0.1", "-f", "alsa", "-i", "hw:1"},
		},
		{
			name:   "negative offset",
			device: "hw:1",
			offset: -50 * time.Millisecond,
			want:   []string{"-itsoffset", "-0.05", "-f", "alsa", "-i", "hw:1"},
		},
		{
			name:   "offset in whole and half seconds",
			device: "hw:1",
			offset: 1500 * time.Millisecond,
			want:   []string{"-itsoffset", "1.5", "-f", "alsa", "-i", "hw:1"},
		},
		{
			name:    "empty device",
			device:  "",
			wantErr: true,
		},
		{
			name:    "empty device with offset",
			device:  "",
			offset:  100 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AudioInputArgs(tt.device, tt.offset)

			if (err != nil) != tt.wantErr {
				t.Fatalf("AudioInputArgs(%q, %v) error = %v, wantErr %v", tt.device, tt.offset, err, tt.wantErr)
			}
			if tt.wantErr {
				if got != nil {
					t.Errorf("AudioInputArgs(%q, %v) = %v, want nil on error", tt.device, tt.offset, got)
				}
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("AudioInputArgs(%q, %v) = %v, want %v", tt.device, tt.offset, got, tt.want)
			}
		})
	}
}

func TestAudioOutputArgs(t *testing.T) {
	want := []string{"-copyts", "-start_at_zero", "-c:a", "aac", "-ac", "2"}

	if got := AudioOutputArgs(); !slices.Equal(got, want) {
		t.Errorf("AudioOutputArgs() = %v, want %v", got, want)
	}
}

func TestVideoSyncArgs(t *testing.T) {
	want := []string{
		"-use_wallclock_as_timestamps", "1",
		"-probesize", "32",
		"-analyzeduration", "0",
		"-fflags", "nobuffer",
	}

	if got := VideoSyncArgs(); !slices.Equal(got, want) {
		t.Errorf("VideoSyncArgs() = %v, want %v", got, want)
	}
}

func TestAudioArgsReturnFreshSlices(t *testing.T) {
	first := AudioOutputArgs()
	first[0] = "changed"
	if AudioOutputArgs()[0] != "-copyts" {
		t.Error("AudioOutputArgs returns a shared slice; changing one result changed the next")
	}

	sync := VideoSyncArgs()
	sync[0] = "changed"
	if VideoSyncArgs()[0] != "-use_wallclock_as_timestamps" {
		t.Error("VideoSyncArgs returns a shared slice; changing one result changed the next")
	}
}
