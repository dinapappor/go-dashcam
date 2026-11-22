package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeALSA(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	procAsound := filepath.Join(root, "proc", "asound")
	devSnd := filepath.Join(root, "dev", "snd")

	for _, dir := range []string{
		filepath.Join(procAsound, "card0"),
		filepath.Join(procAsound, "card1"),
		devSnd,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	links := map[string]string{
		"vc4hdmi": "card0",
		"C93":     "card1",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(procAsound, name)); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{
		"pcmC0D0p",
		"pcmC1D0c",
		"controlC0",
		"controlC1",
	} {
		if err := os.WriteFile(filepath.Join(devSnd, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	return procAsound, devSnd
}

func TestPresent(t *testing.T) {
	procAsound, devSnd := fakeALSA(t)

	tests := []struct {
		device string
		want   bool
	}{
		{"hw:1", true},
		{"hw:1,0", true},
		{"plughw:1", true},
		{"plughw:1,0", true},
		{"hw:1,1", false},
		{"hw:3", false},
		{"hw:0", false},
		{"hw:C93", true},
		{"hw:C93,0", true},
		{"plughw:C93", true},
		{"plughw:CARD=C93,DEV=0", true},
		{"plughw:CARD=C93", true},
		{"hw:CARD=1,DEV=0", true},
		{"hw:CARD=C93,DEV=1", false},
		{"hw:vc4hdmi", false},
		{"hw:Missing", false},
		{"hw:1,x", false},
		{"hw:", false},
		{"plughw:", false},
		{"default", true},
		{"pulse", true},
	}

	for _, tt := range tests {
		t.Run(tt.device, func(t *testing.T) {
			if got := Present(tt.device, procAsound, devSnd); got != tt.want {
				t.Errorf("Present(%q) = %v, want %v", tt.device, got, tt.want)
			}
		})
	}
}

func TestPresentMissingDirectories(t *testing.T) {
	root := t.TempDir()
	procAsound := filepath.Join(root, "proc", "asound")
	devSnd := filepath.Join(root, "dev", "snd")

	tests := []struct {
		device string
		want   bool
	}{
		{"hw:1", false},
		{"hw:C93", false},
		{"default", true},
	}

	for _, tt := range tests {
		t.Run(tt.device, func(t *testing.T) {
			if got := Present(tt.device, procAsound, devSnd); got != tt.want {
				t.Errorf("Present(%q) with no sound devices = %v, want %v", tt.device, got, tt.want)
			}
		})
	}
}
