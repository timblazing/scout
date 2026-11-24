// Package types defines the core domain types for Scout
package types

import "time"

// Device is a device found by a scan. It carries only what discovery knows;
// the user's label and icon live in the names file and are applied when the
// device list is served.
type Device struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname"`
	Vendor   string `json:"vendor"`
	// Type is the inferred device kind (Phone, Printer, TV, and so on), or ""
	// when it could not be determined. It is derived at scan time from the
	// vendor, the hostname and, when service detection is on, the open ports.
	Type string `json:"type,omitempty"`
	// WebUI is true when the device answered on a common web port during an
	// opt-in service probe, meaning it likely serves a dashboard or admin page.
	// Always false when service detection is off.
	WebUI bool `json:"web_ui,omitempty"`
	// Risks lists security concerns found during an opt-in service probe, such as
	// an exposed unencrypted service. Empty when nothing notable was found or the
	// probe is off.
	Risks []string `json:"risks,omitempty"`
}

// Network represents a detected network interface
type Network struct {
	CIDR         string `json:"cidr"`
	Interface    string `json:"interface"`
	FriendlyName string `json:"friendly_name"`
	IP           string `json:"ip"`
	IsTailscale  bool   `json:"is_tailscale"`
	IsWireless   bool   `json:"is_wireless"`
}

// ScanResult represents the outcome of a network scan
type ScanResult struct {
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Devices     []Device  `json:"devices"`
	DeviceCount int       `json:"device_count"`
	Network     string    `json:"network"`
	Scanner     string    `json:"scanner"`
	Duration    float64   `json:"duration"`
	Timestamp   time.Time `json:"timestamp"`
}

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
