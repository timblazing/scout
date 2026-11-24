// Package api implements the REST API and the background scanner behind it
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/291-Group/LAN-Orangutan/internal/config"
	"github.com/291-Group/LAN-Orangutan/internal/network"
	"github.com/291-Group/LAN-Orangutan/internal/scanner"
	"github.com/291-Group/LAN-Orangutan/internal/storage"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

// Handler handles API requests
type Handler struct {
	live    *storage.Live
	names   *storage.Names
	cfg     *config.Config
	scanner *scanner.Scanner

	// scanWG tracks the background scan goroutine so shutdown can wait for it to
	// unwind rather than abandon a scan mid-run.
	scanWG sync.WaitGroup
}

// NewHandler creates a new API handler
func NewHandler(live *storage.Live, names *storage.Names, cfg *config.Config) *Handler {
	return &Handler{
		live:    live,
		names:   names,
		cfg:     cfg,
		scanner: scanner.New(cfg.Scanning.EnableServiceDetection),
	}
}

// device is a device as the dashboard sees it. Vendor and Type are resolved
// here so the UI does not have to reimplement vendor lookup or classification.
type device struct {
	ID       string   `json:"id"`
	IP       string   `json:"ip"`
	MAC      string   `json:"mac"`
	Hostname string   `json:"hostname"`
	Vendor   string   `json:"vendor"`
	Type     string   `json:"type"`
	Label    string   `json:"label"`
	Icon     string   `json:"icon"`
	WebUI    bool     `json:"web_ui"`
	Risks    []string `json:"risks"`
}

// deviceList is the response of GET /api/devices.
type deviceList struct {
	Devices []device `json:"devices"`
	Warning string   `json:"warning,omitempty"`
	// Scanning is true until the first scan cycle has finished, so the UI can
	// tell "nothing found yet" from "nothing here".
	Scanning bool `json:"scanning,omitempty"`
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// CSRF protection for mutating requests
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
		contentType := r.Header.Get("Content-Type")
		xRequestedWith := r.Header.Get("X-Requested-With")

		if !strings.Contains(contentType, "application/json") && xRequestedWith != "XMLHttpRequest" {
			h.error(w, http.StatusForbidden, "CSRF protection: requires Content-Type: application/json or X-Requested-With header")
			return
		}
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "devices":
		h.handleDevices(w, r)
	case strings.HasPrefix(path, "devices/"):
		h.handleDevice(w, r, strings.TrimPrefix(path, "devices/"))
	default:
		h.error(w, http.StatusNotFound, "endpoint not found")
	}
}

// handleDevices handles GET /api/devices
func (h *Handler) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	snapshot := h.live.Snapshot()
	out := deviceList{
		Devices:  make([]device, 0, len(snapshot)),
		Scanning: !h.live.Scanned(),
	}

	for _, key := range storage.SortedKeys(snapshot) {
		d := snapshot[key]
		vendor := scanner.ResolveVendor(d.Vendor, d.MAC)
		deviceType := d.Type
		if deviceType == "" {
			deviceType = scanner.Classify(vendor, d.Hostname, nil)
		}
		name := h.names.Get(key)
		risks := d.Risks
		if risks == nil {
			risks = []string{}
		}
		out.Devices = append(out.Devices, device{
			ID: key, IP: d.IP, MAC: d.MAC, Hostname: d.Hostname,
			Vendor: vendor, Type: deviceType,
			Label: name.Label, Icon: name.Icon,
			WebUI: d.WebUI, Risks: risks,
		})
	}

	networks, _ := network.DetectNetworks()
	out.Warning = network.IsolationWarning(h.configuredNetworks(networks))

	h.success(w, out)
}

// handleDevice handles PATCH /api/devices/{id}, which sets a device's label
// and/or icon. Only a device currently on the network can be edited.
func (h *Handler) handleDevice(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		h.error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Label *string `json:"label"`
		Icon  *string `json:"icon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Icon != nil && *req.Icon != "" && !validDeviceIcon(*req.Icon) {
		h.error(w, http.StatusBadRequest, "invalid icon")
		return
	}
	if req.Label != nil {
		trimmed := strings.TrimSpace(*req.Label)
		if len(trimmed) > 120 {
			h.error(w, http.StatusBadRequest, "name is too long")
			return
		}
		req.Label = &trimmed
	}
	if !h.live.Has(id) {
		h.error(w, http.StatusNotFound, "device not found")
		return
	}
	if err := h.names.Set(id, req.Label, req.Icon); err != nil {
		h.error(w, http.StatusInternalServerError, "failed to save device")
		return
	}
	h.success(w, map[string]string{"message": "device updated"})
}

func validDeviceIcon(icon string) bool {
	parts := strings.SplitN(icon, ":", 2)
	if len(parts) != 2 || (parts[0] != "dashboard" && parts[0] != "lucide") || len(parts[1]) == 0 || len(parts[1]) > 80 {
		return false
	}
	for _, r := range parts[1] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// configuredNetworks applies the user's network settings to a detected list.
func (h *Handler) configuredNetworks(detected []types.Network) []types.Network {
	return network.WithConfigured(detected, network.Filter{
		Configured:     h.cfg.Scanning.Networks,
		Excluded:       h.cfg.Scanning.ExcludeNetworks,
		OnlyConfigured: h.cfg.Scanning.OnlyConfiguredNetworks,
	})
}

// Wait blocks until the background scanner has returned. Call it at shutdown,
// after the context passed to StartBackgroundScanner has been cancelled.
func (h *Handler) Wait() { h.scanWG.Wait() }

// success sends a successful JSON response
func (h *Handler) success(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// error sends an error JSON response
func (h *Handler) error(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.APIResponse{Success: false, Error: message})
}
