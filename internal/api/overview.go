package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/291-Group/LAN-Orangutan/internal/network"
	"github.com/291-Group/LAN-Orangutan/internal/scanner"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

// overviewDevice is a device with the display fields the dashboard needs
// resolved on the server, so the UI does not have to reimplement vendor lookup,
// classification or the online thresholds.
type overviewDevice struct {
	*types.Device
	// Status is "online" (seen in the last 5 minutes), "seen" (last hour) or
	// "offline".
	Status string `json:"status"`
	// Vendor and Type shadow the stored values so records that predate the
	// vendor database or classification still show something useful.
	Vendor string `json:"vendor"`
	Type   string `json:"type,omitempty"`
}

// overview is everything the dashboard shows, in one response, so the UI can
// poll a single endpoint.
type overview struct {
	Stats          types.DeviceStats `json:"stats"`
	Devices        []overviewDevice  `json:"devices"`
	Networks       []types.Network   `json:"networks"`
	NetworkWarning string            `json:"network_warning,omitempty"`
	Flagged        int               `json:"flagged"`
	Moved          int               `json:"moved"`
	LastScan       *time.Time        `json:"last_scan,omitempty"`
	ContinuousScan bool              `json:"continuous_scan"`
	Scanning       bool              `json:"scanning"`
}

// handleOverview handles GET /api/overview.
func (h *Handler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	devices := h.store.GetDevices()
	out := overview{
		Stats:          h.store.GetStats(),
		Devices:        make([]overviewDevice, 0, len(devices)),
		ContinuousScan: h.store.ContinuousScanEnabled(h.cfg.Scanning.ContinuousScan),
	}

	for _, d := range devices {
		if len(d.Risks) > 0 {
			out.Flagged++
		}
		if len(d.AddressHistory) > 0 {
			out.Moved++
		}

		vendor := scanner.ResolveVendor(d.Vendor, d.MAC)
		deviceType := d.Type
		if deviceType == "" {
			deviceType = scanner.Classify(vendor, d.Hostname, nil)
		}

		status := "offline"
		if d.IsRecent() {
			status = "online"
		} else if d.IsOnline() {
			status = "seen"
		}

		out.Devices = append(out.Devices, overviewDevice{
			Device: d,
			Status: status,
			Vendor: vendor,
			Type:   deviceType,
		})
	}

	sort.Slice(out.Devices, func(i, j int) bool {
		return ipSortKey(out.Devices[i].IP) < ipSortKey(out.Devices[j].IP)
	})

	networks, _ := network.DetectNetworks()
	out.Networks = network.WithConfigured(networks, network.Filter{
		Configured:     h.cfg.Scanning.Networks,
		Excluded:       h.cfg.Scanning.ExcludeNetworks,
		OnlyConfigured: h.cfg.Scanning.OnlyConfiguredNetworks,
	})
	if out.Networks == nil {
		out.Networks = []types.Network{}
	}
	out.NetworkWarning = network.IsolationWarning(out.Networks)

	if last := h.store.GetMostRecentScan(); !last.IsZero() {
		out.LastScan = &last
	}

	h.jobMu.Lock()
	out.Scanning = h.job != nil && h.job.isRunning()
	h.jobMu.Unlock()

	h.success(w, out)
}

// ipSortKey orders IPv4 addresses numerically and puts anything else after
// them.
func ipSortKey(ip string) int64 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 1 << 40
	}
	var key int64
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 1 << 40
		}
		key = key<<8 | int64(n&0xff)
	}
	return key
}
