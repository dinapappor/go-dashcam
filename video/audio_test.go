package video

import (
	"slices"
	"testing"
)

func TestAudioInputArgs(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		want    []string
		wantErr bool
	}{
		{
			name:   "hw device",
			device: "hw:1",
			want:   []string{"-thread_queue_size", "512", "-f", "alsa", "-i", "hw:1"},
		},
		{
			name:   "plughw with subdevice",
			device: "plughw:1,0",
			want:   []string{"-thread_queue_size", "512", "-f", "alsa", "-i", "plughw:1,0"},
		},
		{
			name:   "default device",
			device: "default",
			want:   []string{"-thread_queue_size", "512", "-f", "alsa", "-i", "default"},
		},
		{
			name:    "empty device",
			device:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AudioInputArgs(tt.device)

			if (err != nil) != tt.wantErr {
				t.Fatalf("AudioInputArgs(%q) error = %v, wantErr %v", tt.device, err, tt.wantErr)
			}
			if tt.wantErr {
				if got != nil {
					t.Errorf("AudioInputArgs(%q) = %v, want nil on error", tt.device, got)
				}
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("AudioInputArgs(%q) = %v, want %v", tt.device, got, tt.want)
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
