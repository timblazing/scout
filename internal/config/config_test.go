package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes body to a temporary config file and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

func TestDefaults(t *testing.T) {
	cfg := Default()

	// Reachable from the network, because that is how this is normally used.
	if cfg.IsLoopbackBind() {
		t.Fatalf("default bind address %q should accept connections from the network", cfg.Server.BindAddress)
	}
	if cfg.Scanning.ScanInterval != 60 {
		t.Errorf("default scan interval = %d, want 60", cfg.Scanning.ScanInterval)
	}
}

func TestMissingFileFallsBackToDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err != nil {
		t.Fatalf("a missing config file should not be an error: %v", err)
	}
	if cfg.Server.Port != Default().Server.Port {
		t.Errorf("port = %d, want the default", cfg.Server.Port)
	}
}

func TestLoadSettings(t *testing.T) {
	path := writeConfig(t, `
[server]
port = 8080
bind_address = 127.0.0.1
; settings from older versions are ignored
password = hunter2

[scanning]
scan_interval = 120
networks = 10.0.5.0/24, 192.168.10.0/24
exclude_networks = 10.99.0.0/16
only_configured_networks = true
enable_service_detection = yes
continuous_scan = false

[storage]
data_dir = /srv/scout
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.BindAddress != "127.0.0.1" {
		t.Errorf("bind_address = %q, want 127.0.0.1", cfg.Server.BindAddress)
	}
	if cfg.Scanning.ScanInterval != 120 {
		t.Errorf("scan_interval = %d, want 120", cfg.Scanning.ScanInterval)
	}
	if len(cfg.Scanning.Networks) != 2 || cfg.Scanning.Networks[1] != "192.168.10.0/24" {
		t.Errorf("networks = %v", cfg.Scanning.Networks)
	}
	if len(cfg.Scanning.ExcludeNetworks) != 1 || cfg.Scanning.ExcludeNetworks[0] != "10.99.0.0/16" {
		t.Errorf("exclude_networks = %v", cfg.Scanning.ExcludeNetworks)
	}
	if !cfg.Scanning.OnlyConfiguredNetworks || !cfg.Scanning.EnableServiceDetection {
		t.Error("only_configured_networks and enable_service_detection should be true")
	}
	if cfg.Storage.DataDir != "/srv/scout" {
		t.Errorf("data_dir = %q", cfg.Storage.DataDir)
	}
}

func TestIsLoopbackBind(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.53", true},
		{"localhost", true},
		{"LOCALHOST", true},
		{"::1", true},
		{"[::1]", true},
		{"0.0.0.0", false},
		{"192.168.1.10", false},
		{"::", false},
		{"", false}, // empty means every interface
		{"not-an-address", false},
	}

	for _, tt := range tests {
		cfg := Default()
		cfg.Server.BindAddress = tt.addr
		if got := cfg.IsLoopbackBind(); got != tt.want {
			t.Errorf("IsLoopbackBind(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

func TestEnvOverridesConfigFile(t *testing.T) {
	path := writeConfig(t, `
[server]
port = 8080
bind_address = 127.0.0.1
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	t.Setenv("ORANGUTAN_PORT", "9999")
	t.Setenv("ORANGUTAN_BIND_ADDRESS", "0.0.0.0")
	t.Setenv("ORANGUTAN_SCAN_INTERVAL", "30")
	t.Setenv("ORANGUTAN_DATA_DIR", "/tmp/scout-data")
	cfg.ApplyEnv()

	if cfg.Server.Port != 9999 {
		t.Errorf("port = %d, want the environment value 9999", cfg.Server.Port)
	}
	if cfg.Server.BindAddress != "0.0.0.0" {
		t.Errorf("bind_address = %q, want the environment value", cfg.Server.BindAddress)
	}
	if cfg.Scanning.ScanInterval != 30 {
		t.Errorf("scan_interval = %d, want 30", cfg.Scanning.ScanInterval)
	}
	if got, want := cfg.NamesFile(), filepath.Join("/tmp/scout-data", "names.json"); got != want {
		t.Errorf("NamesFile() = %q, want %q", got, want)
	}
}

func TestEnvLeavesUnsetValuesAlone(t *testing.T) {
	cfg := Default()
	original := cfg.Server.Port

	cfg.ApplyEnv() // nothing set

	if cfg.Server.Port != original {
		t.Errorf("port changed to %d with no environment variable set", cfg.Server.Port)
	}
	if cfg.Server.BindAddress != Default().Server.BindAddress {
		t.Error("bind address should be untouched with no environment variable set")
	}
}

func TestCommentsAndBlankLinesIgnored(t *testing.T) {
	path := writeConfig(t, `
# a comment
; another comment

[server]
port = 4242
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 4242 {
		t.Errorf("port = %d, want 4242", cfg.Server.Port)
	}
}

func TestNormalizeClampsNonPositiveScanInterval(t *testing.T) {
	c := Default()
	c.Scanning.ScanInterval = 0

	notes := c.Normalize()

	if c.Scanning.ScanInterval != defaultScanInterval {
		t.Errorf("scan_interval = %d, want clamped to %d", c.Scanning.ScanInterval, defaultScanInterval)
	}
	if len(notes) == 0 {
		t.Error("expected a warning note about scan_interval, got none")
	}
}

func TestNormalizeLeavesValidValuesAlone(t *testing.T) {
	c := Default()

	notes := c.Normalize()

	if len(notes) != 0 {
		t.Errorf("expected no warnings for default config, got %v", notes)
	}
	if c.Scanning.ScanInterval != defaultScanInterval {
		t.Error("Normalize changed an already-valid value")
	}
}
