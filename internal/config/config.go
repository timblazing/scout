// Package config handles configuration loading and management
package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/291-Group/LAN-Orangutan/internal/network"
)

// GetDefaultDataDir returns the appropriate default data directory for the current OS
func GetDefaultDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: use Application Support
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/lan-orangutan"
		}
		return filepath.Join(home, "Library", "Application Support", "lan-orangutan")

	case "windows":
		// Windows: use AppData
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "C:\\lan-orangutan"
			}
			return filepath.Join(home, "AppData", "Roaming", "lan-orangutan")
		}
		return filepath.Join(appData, "lan-orangutan")

	default:
		// Linux and others
		// Use /var/lib if running as root, otherwise use user's local share
		if os.Getuid() == 0 {
			return "/var/lib/lan-orangutan"
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/lan-orangutan"
		}
		return filepath.Join(home, ".local", "share", "lan-orangutan")
	}
}

// GetDefaultConfigFile returns the appropriate default config file path for the current OS
func GetDefaultConfigFile() string {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/lan-orangutan/config.ini"
		}
		return filepath.Join(home, "Library", "Application Support", "lan-orangutan", "config.ini")

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "C:\\lan-orangutan\\config.ini"
		}
		return filepath.Join(appData, "lan-orangutan", "config.ini")

	default:
		// Linux: use /etc if running as root, otherwise use user config
		if os.Getuid() == 0 {
			return "/etc/lan-orangutan/config.ini"
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/lan-orangutan/config.ini"
		}
		return filepath.Join(home, ".config", "lan-orangutan", "config.ini")
	}
}

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Scanning ScanningConfig
	Storage  StorageConfig
}

// ServerConfig holds web server settings. Scout does not authenticate dashboard
// or API requests.
type ServerConfig struct {
	Port        int
	BindAddress string
}

// ScanningConfig holds scanner settings
type ScanningConfig struct {
	// ScanInterval is how many seconds pass between background scans of the
	// detected networks. The server always scans on its own; there is no manual
	// scan.
	ScanInterval int

	// EnableServiceDetection turns on a light probe of a few well-known ports
	// during a scan, used only to identify devices more precisely (a printer, a
	// NAS, a device with a web interface). Off by default so the scan stays a
	// quiet ping sweep unless the user opts in.
	EnableServiceDetection bool

	// OnlyConfiguredNetworks, when true, scans only the networks declared in
	// Networks and ignores everything auto-detection finds. Useful on a host
	// with many Docker bridges, VLANs or routed interfaces where the automatic
	// list is noisy.
	OnlyConfiguredNetworks bool

	// ExcludeNetworks are CIDRs to drop from the auto-detected list, for real
	// networks the user does not want scanned. A blocklist that complements
	// Networks (the allowlist): keep auto-detection but hide a few subnets.
	ExcludeNetworks []string

	// Networks are CIDRs the user has declared explicitly, for cases where
	// automatic detection cannot see the right network. A container only sees
	// Docker's private network, so without this it can never scan the LAN.
	Networks []string
}

// StorageConfig holds data storage settings
type StorageConfig struct {
	DataDir string
}

// defaultScanInterval is also used by Normalize to repair an out-of-range
// value rather than let it silently disable the background scanner.
const defaultScanInterval = 60

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 291,
			// Reachable from the network, because this is usually installed on
			// a server or a Pi and opened from another machine. The dashboard
			// has no login, so keep it on a trusted network.
			BindAddress: "0.0.0.0",
		},
		Scanning: ScanningConfig{
			ScanInterval: defaultScanInterval,
		},
		Storage: StorageConfig{
			DataDir: GetDefaultDataDir(),
		},
	}
}

// Load reads configuration from an INI file. Keys it does not know, such as
// settings from older versions, are ignored.
func Load(path string) (*Config, error) {
	cfg := Default()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return defaults if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	var currentSection string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		cfg.setValue(currentSection, key, value)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return cfg, nil
}

// setValue sets a configuration value based on section and key
func (c *Config) setValue(section, key, value string) {
	switch section {
	case "server":
		switch key {
		case "port":
			if v, err := strconv.Atoi(value); err == nil {
				c.Server.Port = v
			}
		case "bind_address":
			c.Server.BindAddress = value
		}
	case "scanning":
		switch key {
		case "scan_interval":
			if v, err := strconv.Atoi(value); err == nil {
				c.Scanning.ScanInterval = v
			}
		case "enable_service_detection":
			c.Scanning.EnableServiceDetection = parseBool(value)
		case "only_configured_networks":
			c.Scanning.OnlyConfiguredNetworks = parseBool(value)
		case "networks":
			c.Scanning.Networks = network.ParseNetworkList(value)
		case "exclude_networks":
			c.Scanning.ExcludeNetworks = network.ParseNetworkList(value)
		}
	case "storage":
		if key == "data_dir" {
			c.Storage.DataDir = value
		}
	}
}

// ApplyEnv overlays settings from environment variables onto c.
//
// Environment variables take precedence over the config file, which makes the
// app configurable in a container without mounting a config file. Command line
// flags still win over both.
func (c *Config) ApplyEnv() {
	if v := os.Getenv("ORANGUTAN_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Server.Port = n
		}
	}
	if v := os.Getenv("ORANGUTAN_BIND_ADDRESS"); v != "" {
		c.Server.BindAddress = v
	}
	if v := os.Getenv("ORANGUTAN_DATA_DIR"); v != "" {
		c.Storage.DataDir = v
	}
	if v := os.Getenv("ORANGUTAN_NETWORKS"); v != "" {
		c.Scanning.Networks = network.ParseNetworkList(v)
	}
	if v := os.Getenv("ORANGUTAN_ENABLE_SERVICE_DETECTION"); v != "" {
		c.Scanning.EnableServiceDetection = parseBool(v)
	}
	if v := os.Getenv("ORANGUTAN_ONLY_CONFIGURED_NETWORKS"); v != "" {
		c.Scanning.OnlyConfiguredNetworks = parseBool(v)
	}
	if v := os.Getenv("ORANGUTAN_EXCLUDE_NETWORKS"); v != "" {
		c.Scanning.ExcludeNetworks = network.ParseNetworkList(v)
	}
	if v := os.Getenv("ORANGUTAN_SCAN_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Scanning.ScanInterval = n
		}
	}
}

// Normalize repairs an out-of-range scan interval, clamping it back to the
// default, and returns a human-readable note for each value it had to correct.
// A non-positive scan_interval would otherwise silently disable the background
// scanner, so a config typo cannot quietly leave the device list empty. Call it
// after loading the config from file and environment.
func (c *Config) Normalize() []string {
	var notes []string

	if c.Scanning.ScanInterval <= 0 {
		notes = append(notes, fmt.Sprintf(
			"scan_interval was %d; using %d so the device list keeps updating",
			c.Scanning.ScanInterval, defaultScanInterval))
		c.Scanning.ScanInterval = defaultScanInterval
	}

	return notes
}

// IsLoopbackBind reports whether the configured bind address only accepts
// connections from the machine the app is running on.
//
// An empty bind address means every interface, which is not loopback.
func (c *Config) IsLoopbackBind() bool {
	return isLoopbackAddress(c.Server.BindAddress)
}

func isLoopbackAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	if strings.EqualFold(addr, "localhost") {
		return true
	}
	// Strip brackets from IPv6 literals such as [::1].
	addr = strings.TrimPrefix(strings.TrimSuffix(addr, "]"), "[")

	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// DevicesFile returns the path of the device list older versions kept. Scout
// only reads it, once, to import names and icons into the names file.
func (c *Config) DevicesFile() string {
	return filepath.Join(c.Storage.DataDir, "devices.json")
}

// NamesFile returns the path of the file holding the user's device names and
// icons, the only thing Scout persists.
func (c *Config) NamesFile() string {
	return filepath.Join(c.Storage.DataDir, "names.json")
}

// parseBool parses common boolean representations
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1" || s == "on"
}
