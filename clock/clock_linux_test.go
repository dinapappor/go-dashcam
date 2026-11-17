package clock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynced(t *testing.T) {
	dir := t.TempDir()

	syncedFile := filepath.Join(dir, "synchronized")
	if err := os.WriteFile(syncedFile, nil, 0o644); err != nil {
		t.Fatalf("create marker file: %v", err)
	}

	unreadableDir := filepath.Join(dir, "locked")
	if err := os.Mkdir(unreadableDir, 0o000); err != nil {
		t.Fatalf("create locked dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(unreadableDir, 0o755) })

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "marker file exists", path: syncedFile, want: true},
		{name: "marker file missing", path: filepath.Join(dir, "missing"), want: false},
		{name: "parent directory missing", path: filepath.Join(dir, "nope", "synchronized"), want: false},
		{name: "empty path", path: "", want: false},
		{name: "existing directory counts as synced", path: dir, want: true},
	}

	if os.Geteuid() != 0 {
		tests = append(tests, struct {
			name string
			path string
			want bool
		}{name: "file behind unreadable directory", path: filepath.Join(unreadableDir, "synchronized"), want: false})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Synced(tt.path); got != tt.want {
				t.Errorf("Synced(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSyncedRealSystem(t *testing.T) {
	path := "/run/systemd/timesync/synchronized"
	_, err := os.Stat(path)
	want := err == nil

	if got := Synced(path); got != want {
		t.Errorf("Synced(%q) = %v, but os.Stat says exists=%v", path, got, want)
	}
}
