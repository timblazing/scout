package storage

import (
	"testing"

	"github.com/291-Group/LAN-Orangutan/internal/types"
)

func dev(ip, mac string) types.Device { return types.Device{IP: ip, MAC: mac} }

func TestKey(t *testing.T) {
	if got := Key("AA:BB:CC:00:11:22", "10.0.0.5"); got != "aa:bb:cc:00:11:22" {
		t.Errorf("Key with MAC = %q", got)
	}
	if got := Key("", "10.0.0.5"); got != "ip:10.0.0.5" {
		t.Errorf("Key without MAC = %q", got)
	}
}

func TestMissCountRemoval(t *testing.T) {
	l := NewLive()
	a := dev("10.0.0.2", "aa:aa:aa:aa:aa:01")
	b := dev("10.0.0.3", "aa:aa:aa:aa:aa:02")

	l.Apply([]types.Device{a, b}, nil, nil)
	if got := len(l.Snapshot()); got != 2 {
		t.Fatalf("present: %d devices, want 2", got)
	}

	// b misses once: still listed.
	l.Apply([]types.Device{a}, nil, nil)
	if !l.Has(Key(b.MAC, b.IP)) {
		t.Fatal("a device missed once must still be shown")
	}

	// b reappears: the count resets, so one more miss keeps it.
	l.Apply([]types.Device{a, b}, nil, nil)
	l.Apply([]types.Device{a}, nil, nil)
	if !l.Has(Key(b.MAC, b.IP)) {
		t.Fatal("a device that came back must start counting from zero")
	}

	// Missed twice in a row: gone.
	l.Apply([]types.Device{a}, nil, nil)
	if l.Has(Key(b.MAC, b.IP)) {
		t.Fatal("a device missed twice in a row must be removed")
	}
	if !l.Has(Key(a.MAC, a.IP)) {
		t.Fatal("the device still found must stay")
	}
}

func TestScannedAfterFirstCycle(t *testing.T) {
	l := NewLive()
	if l.Scanned() {
		t.Fatal("a new set has not scanned yet")
	}
	l.Apply(nil, nil, nil)
	if !l.Scanned() {
		t.Fatal("a completed cycle marks the set scanned")
	}
	l2 := NewLive()
	l2.MarkScanned()
	if !l2.Scanned() {
		t.Fatal("MarkScanned should mark the set scanned")
	}
}

func TestRefreshKeepsKnownDetailsAndFollowsIP(t *testing.T) {
	l := NewLive()
	mac := "aa:aa:aa:aa:aa:01"
	l.Apply([]types.Device{{IP: "10.0.0.2", MAC: mac, Hostname: "printer", Vendor: "HP", Type: "Printer", WebUI: true}}, nil, nil)

	// New address, no hostname/vendor this time, no web UI any more.
	l.Apply([]types.Device{{IP: "10.0.0.9", MAC: mac}}, nil, nil)
	got := l.Snapshot()[mac]
	if got.IP != "10.0.0.9" {
		t.Errorf("IP = %s, want the new address", got.IP)
	}
	if got.Hostname != "printer" || got.Vendor != "HP" || got.Type != "Printer" {
		t.Errorf("known details were wiped: %+v", got)
	}
	if got.WebUI {
		t.Error("probe fields must reflect the latest scan")
	}
	if len(l.Snapshot()) != 1 {
		t.Error("a device that changed address is still one row")
	}
}

func TestMACIsCaseInsensitive(t *testing.T) {
	l := NewLive()
	l.Apply([]types.Device{dev("10.0.0.2", "AA:AA:AA:AA:AA:01")}, nil, nil)
	l.Apply([]types.Device{dev("10.0.0.2", "aa:aa:aa:aa:aa:01")}, nil, nil)
	if got := len(l.Snapshot()); got != 1 {
		t.Fatalf("%d rows, want 1", got)
	}
}

func TestDedupeSharedMAC(t *testing.T) {
	l := NewLive()
	mac := "aa:aa:aa:aa:aa:01"
	l.Apply([]types.Device{
		{IP: "10.0.0.20", MAC: mac, Hostname: "second"},
		{IP: "10.0.0.4", MAC: mac},
	}, nil, nil)

	snap := l.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("%d rows, want 1", len(snap))
	}
	got := snap[mac]
	if got.IP != "10.0.0.4" {
		t.Errorf("IP = %s, want the lowest address", got.IP)
	}
	if got.Hostname != "second" {
		t.Errorf("hostname = %q, want it filled from the duplicate", got.Hostname)
	}
}

func TestIPv6EnrichesButNeverCreates(t *testing.T) {
	l := NewLive()
	mac := "aa:aa:aa:aa:aa:01"
	ipv6 := []types.Device{
		{IP: "2001:db8::1", MAC: "AA:AA:AA:AA:AA:01", Hostname: "laptop", Vendor: "Apple"},
		{IP: "2001:db8::2", MAC: "02:00:00:00:00:99", Hostname: "stranger"}, // no IPv4 device has this MAC
		{IP: "2001:db8::3"}, // no MAC at all
	}
	l.Apply([]types.Device{dev("10.0.0.2", mac)}, ipv6, nil)

	snap := l.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("%d rows, want 1 (IPv6 must not create rows)", len(snap))
	}
	got := snap[mac]
	if got.IP != "10.0.0.2" {
		t.Errorf("displayed IP = %s, want the IPv4 address", got.IP)
	}
	if got.Hostname != "laptop" || got.Vendor != "Apple" {
		t.Errorf("IPv6 neighbour should enrich: %+v", got)
	}
}

func TestIPv6IgnoredAsPrimary(t *testing.T) {
	l := NewLive()
	l.Apply([]types.Device{{IP: "2001:db8::1", MAC: "aa:aa:aa:aa:aa:01"}}, nil, nil)
	if len(l.Snapshot()) != 0 {
		t.Fatal("an IPv6 sweep result must not create a row")
	}
}

func TestMDNSEnrichesByIPButNeverCreates(t *testing.T) {
	l := NewLive()
	mdns := []types.Device{
		{IP: "10.0.0.2", Hostname: "tv", Type: "TV"},
		{IP: "10.0.0.77", Hostname: "ghost"},
	}
	l.Apply([]types.Device{dev("10.0.0.2", "aa:aa:aa:aa:aa:01")}, nil, mdns)

	snap := l.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("%d rows, want 1 (mDNS must not create rows)", len(snap))
	}
	if got := snap["aa:aa:aa:aa:aa:01"]; got.Hostname != "tv" || got.Type != "TV" {
		t.Errorf("mDNS should fill blanks: %+v", got)
	}
}

func TestTailscaleRangeDropped(t *testing.T) {
	l := NewLive()
	l.Apply([]types.Device{
		{IP: "100.64.0.1"},
		{IP: "100.100.1.2", Hostname: "peer"},
		{IP: "100.128.0.1"}, // just outside 100.64.0.0/10
		dev("192.168.1.5", ""),
	}, nil, nil)

	snap := l.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("%d rows, want 2: %v", len(snap), snap)
	}
	if _, ok := snap["ip:100.64.0.1"]; ok {
		t.Error("100.64.0.1 is a Tailscale address and must be dropped")
	}
	if _, ok := snap["ip:100.100.1.2"]; ok {
		t.Error("100.100.1.2 is a Tailscale address and must be dropped")
	}
	if _, ok := snap["ip:100.128.0.1"]; !ok {
		t.Error("100.128.0.1 is outside the Tailscale range and must be kept")
	}
}

func TestSortedKeysByIPNumerically(t *testing.T) {
	l := NewLive()
	l.Apply([]types.Device{dev("10.0.0.100", ""), dev("10.0.0.9", ""), dev("10.0.0.20", "")}, nil, nil)
	keys := SortedKeys(l.Snapshot())
	want := []string{"ip:10.0.0.9", "ip:10.0.0.20", "ip:10.0.0.100"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}
