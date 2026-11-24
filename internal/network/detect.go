// Package network handles network interface detection and utilities
package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/291-Group/LAN-Orangutan/internal/types"
)

// DetectNetworks discovers available network interfaces and their CIDRs
// Uses Go's standard library for cross-platform compatibility
func DetectNetworks() ([]types.Network, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var networks []types.Network
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6
			if ipNet.IP.To4() == nil {
				continue
			}

			// Calculate network CIDR
			ones, _ := ipNet.Mask.Size()
			networkIP := ipNet.IP.Mask(ipNet.Mask)
			cidr := fmt.Sprintf("%s/%d", networkIP.String(), ones)

			network := types.Network{
				CIDR:         cidr,
				Interface:    iface.Name,
				FriendlyName: getFriendlyName(iface.Name, ipNet.IP),
				IP:           ipNet.IP.String(),
				IsTailscale:  isTailscaleInterface(iface.Name, ipNet.IP),
				IsWireless:   isWirelessInterface(iface.Name),
			}
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// getFriendlyName returns a user-friendly name for an interface
func getFriendlyName(ifname string, ip net.IP) string {
	switch {
	case isTailscaleInterface(ifname, ip):
		return "Tailscale VPN"
	case strings.HasPrefix(ifname, "wlan") || strings.HasPrefix(ifname, "wlp"):
		return "Wi-Fi"
	case strings.HasPrefix(ifname, "eth") || strings.HasPrefix(ifname, "enp") || strings.HasPrefix(ifname, "eno"):
		return "Ethernet"
	// macOS interfaces
	case ifname == "en0":
		return "Wi-Fi" // Usually Wi-Fi on macOS
	case strings.HasPrefix(ifname, "en"):
		return "Ethernet" // Other en* interfaces are usually Ethernet
	case strings.HasPrefix(ifname, "bridge"):
		return "Bridge"
	case strings.HasPrefix(ifname, "br"):
		return "Bridge"
	case strings.HasPrefix(ifname, "docker"):
		return "Docker"
	case strings.HasPrefix(ifname, "veth"):
		return "Virtual Ethernet"
	case strings.HasPrefix(ifname, "virbr"):
		return "Virtual Bridge"
	case strings.HasPrefix(ifname, "utun"):
		return "VPN Tunnel"
	case strings.HasPrefix(ifname, "awdl"):
		return "Apple Wireless Direct"
	case strings.HasPrefix(ifname, "llw"):
		return "Low Latency WLAN"
	default:
		return ifname
	}
}

// tailscaleCGNAT is the address range Tailscale assigns to every node.
var tailscaleCGNAT = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

// IsTailscaleIP reports whether ip is in the range Tailscale assigns to nodes
// (the carrier-grade NAT block, 100.64.0.0/10).
func IsTailscaleIP(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && tailscaleCGNAT.Contains(v4)
}

// Scannable reports whether a network is worth sweeping for devices. Tailscale
// addresses and single-address networks (a /32, such as a VPN client address)
// have no neighbours to find, so Scout leaves them out entirely.
func Scannable(cidr string) bool {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil || ip.To4() == nil {
		return false
	}
	if ones, bits := ipNet.Mask.Size(); ones == bits {
		return false
	}
	return !IsTailscaleIP(ip)
}

// isTailscaleInterface returns true if the interface is a Tailscale interface.
//
// The interface name alone is not enough to tell: Linux uses "tailscale0", but
// macOS uses a utun device whose number depends on how many tunnels happen to
// exist, so it is not always utun4. Fall back to checking whether the address
// sits in the range Tailscale assigns, which is what actually identifies it.
// The range is only trusted on a tunnel interface, since it is shared with
// carrier-grade NAT that an ISP could legitimately hand out on a real NIC.
func isTailscaleInterface(ifname string, ip net.IP) bool {
	if strings.HasPrefix(strings.ToLower(ifname), "tailscale") {
		return true
	}
	if !strings.HasPrefix(ifname, "utun") && !strings.HasPrefix(ifname, "tun") {
		return false
	}
	return ip != nil && IsTailscaleIP(ip)
}

// isWirelessInterface returns true if the interface is a wireless interface
func isWirelessInterface(ifname string) bool {
	// Linux wireless interfaces
	if strings.HasPrefix(ifname, "wlan") || strings.HasPrefix(ifname, "wlp") {
		return true
	}
	// macOS - en0 is usually Wi-Fi
	if ifname == "en0" {
		return true
	}
	return false
}

// ValidateCIDR checks if a CIDR string is valid
func ValidateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}
