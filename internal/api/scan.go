package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/291-Group/LAN-Orangutan/internal/network"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

// networkTimeout bounds the sweep of a single network.
const networkTimeout = 5 * time.Minute

// StartBackgroundScanner scans the detected networks right away and then every
// interval, updating the live device list after each full cycle. It stops when
// ctx is cancelled; Wait blocks until it has. Cycles never overlap: a tick that
// arrives while a scan is still running is dropped.
func (h *Handler) StartBackgroundScanner(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}

	h.scanWG.Add(1)
	go func() {
		defer h.scanWG.Done()

		h.scanCycle(ctx)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.scanCycle(ctx)
			}
		}
	}()
}

// scanTargets lists the networks to sweep: detected and configured networks,
// minus Tailscale and single-address networks, which Scout leaves out.
func (h *Handler) scanTargets() []string {
	detected, _ := network.DetectNetworks()
	var targets []string
	for _, n := range h.configuredNetworks(detected) {
		if network.Scannable(n.CIDR) {
			targets = append(targets, n.CIDR)
		}
	}
	return targets
}

// scanCycle sweeps every target network, adds the supplemental IPv6 and mDNS
// sources, and replaces the live set with the result. A cycle where no network
// could be scanned leaves the list as it was, so a missing nmap or a dropped
// interface does not empty the dashboard.
func (h *Handler) scanCycle(ctx context.Context) {
	var primary []types.Device
	scanned := 0
	targets := h.scanTargets()

	for _, cidr := range targets {
		if ctx.Err() != nil {
			return
		}
		netCtx, cancel := context.WithTimeout(ctx, networkTimeout)
		result, err := h.scanner.Scan(netCtx, cidr)
		cancel()
		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "scan %s: %v\n", cidr, err)
		case !result.Success:
			fmt.Fprintf(os.Stderr, "scan %s: %s\n", cidr, result.Error)
		default:
			primary = append(primary, result.Devices...)
			scanned++
		}
	}

	if ctx.Err() != nil {
		return
	}
	if scanned == 0 {
		if len(targets) == 0 {
			fmt.Fprintln(os.Stderr, "scan: no networks to scan")
		}
		h.live.MarkScanned()
		return
	}

	ipv6 := h.scanner.DiscoverIPv6(ctx)
	mdns := h.scanner.DiscoverMDNS(ctx)
	if ctx.Err() != nil {
		return
	}
	h.live.Apply(primary, ipv6, mdns)
}
