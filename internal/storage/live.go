package storage

import (
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/291-Group/LAN-Orangutan/internal/network"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

// maxMisses is how many scan cycles in a row a device may be absent from before
// it is dropped. Two, not one, so a single dropped ARP reply does not make a
// device flicker out of the list.
const maxMisses = 2

// Key is a device's identity: its lowercase MAC when it has one, otherwise
// "ip:" and its IP. A MAC survives a change of address; an IP is all there is
// for a device whose MAC could not be read (a scan without root).
func Key(mac, ip string) string {
	if mac = strings.ToLower(strings.TrimSpace(mac)); mac != "" {
		return mac
	}
	return "ip:" + ip
}

type liveDevice struct {
	types.Device
	misses int
}

// Live is the set of devices currently on the network. It lives in memory only:
// each full scan cycle replaces it with what was found, so nothing offline is
// ever listed.
type Live struct {
	mu      sync.RWMutex
	devices map[string]*liveDevice
	scanned bool
}

// NewLive returns an empty live set that has not completed a scan yet.
func NewLive() *Live {
	return &Live{devices: make(map[string]*liveDevice)}
}

// Scanned reports whether a first scan cycle has completed.
func (l *Live) Scanned() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.scanned
}

// MarkScanned records that a scan cycle completed without changing the set, for
// a cycle that found nothing it could trust (every network failed to scan).
func (l *Live) MarkScanned() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.scanned = true
}

// Has reports whether key identifies a device currently in the set.
func (l *Live) Has(key string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.devices[key]
	return ok
}

// Snapshot returns copies of the devices currently in the set, keyed by
// identity, so callers can read them without racing a later scan.
func (l *Live) Snapshot() map[string]types.Device {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]types.Device, len(l.devices))
	for k, d := range l.devices {
		cp := d.Device
		cp.Risks = append([]string(nil), d.Risks...)
		out[k] = cp
	}
	return out
}

// Apply folds the results of one full scan cycle into the set.
//
// primary holds the IPv4 devices the network sweeps found. Only these create
// rows. ipv6 and mdns are supplemental: IPv6 neighbours enrich the device with
// the same MAC, mDNS answers enrich the device with the same IP, and neither
// ever adds a row of its own (neighbour caches and mDNS are full of privacy
// addresses and stale entries).
//
// A device found has its details refreshed and its miss count reset. One not
// found misses; after maxMisses in a row it is removed.
func (l *Live) Apply(primary, ipv6, mdns []types.Device) {
	found := collapse(primary)

	byIP := make(map[string]string, len(found))
	for key, d := range found {
		byIP[d.IP] = key
	}

	for _, s := range ipv6 {
		if s.MAC == "" {
			continue
		}
		if d, ok := found[Key(s.MAC, "")]; ok {
			fillBlanks(d, s)
		}
	}
	for _, s := range mdns {
		if key, ok := byIP[s.IP]; ok {
			fillBlanks(found[key], s)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.scanned = true

	for key, f := range found {
		cur, ok := l.devices[key]
		if !ok {
			l.devices[key] = &liveDevice{Device: *f}
			continue
		}
		prev := cur.Device
		cur.Device = *f
		// A scan that learned less this time (no reverse DNS answer, no vendor)
		// does not wipe what is already known.
		fillBlanks(&cur.Device, prev)
		cur.misses = 0
	}
	for key, cur := range l.devices {
		if _, ok := found[key]; ok {
			continue
		}
		cur.misses++
		if cur.misses >= maxMisses {
			delete(l.devices, key)
		}
	}
}

// collapse reduces sweep results to one device per identity: IPv4 only, nothing
// in the Tailscale range, and one row where several addresses share a MAC (the
// lowest address wins, so the row does not flip between scans).
func collapse(primary []types.Device) map[string]*types.Device {
	found := make(map[string]*types.Device, len(primary))
	for _, d := range primary {
		ip := net.ParseIP(d.IP)
		if ip == nil || ip.To4() == nil || network.IsTailscaleIP(ip) {
			continue
		}
		d.MAC = strings.ToLower(d.MAC)
		key := Key(d.MAC, d.IP)

		other, dup := found[key]
		if !dup {
			cp := d
			found[key] = &cp
			continue
		}
		keep, extra := d, *other
		if ipSortKey(extra.IP) < ipSortKey(keep.IP) {
			keep, extra = extra, keep
		}
		fillBlanks(&keep, extra)
		found[key] = &keep
	}
	return found
}

// fillBlanks copies identity details from src into dst where dst has none.
func fillBlanks(dst *types.Device, src types.Device) {
	if dst.Hostname == "" {
		dst.Hostname = src.Hostname
	}
	if dst.Vendor == "" {
		dst.Vendor = src.Vendor
	}
	if dst.Type == "" {
		dst.Type = src.Type
	}
}

// ipSortKey orders IPv4 addresses numerically and puts anything else after them.
func ipSortKey(ip string) int64 {
	v4 := net.ParseIP(ip).To4()
	if v4 == nil {
		return 1 << 40
	}
	return int64(v4[0])<<24 | int64(v4[1])<<16 | int64(v4[2])<<8 | int64(v4[3])
}

// SortedKeys returns the keys of a snapshot ordered by IP address, then key.
func SortedKeys(snapshot map[string]types.Device) []string {
	keys := make([]string, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := ipSortKey(snapshot[keys[i]].IP), ipSortKey(snapshot[keys[j]].IP)
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})
	return keys
}
