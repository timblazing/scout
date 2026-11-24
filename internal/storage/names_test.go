package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func str(s string) *string { return &s }

func TestNamesPersistAcrossReopen(t *testing.T) {
	file := filepath.Join(t.TempDir(), "names.json")
	n, err := OpenNames(file, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Set("aa:bb", str("Kitchen"), str("lucide:tv")); err != nil {
		t.Fatal(err)
	}
	if err := n.Set("aa:bb", str("Living room"), nil); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenNames(file, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get("aa:bb"); got.Label != "Living room" || got.Icon != "lucide:tv" {
		t.Fatalf("got %+v", got)
	}
}

func TestNamesClearRemovesEntry(t *testing.T) {
	file := filepath.Join(t.TempDir(), "names.json")
	n, _ := OpenNames(file, "")
	n.Set("k", str("x"), nil)
	n.Set("k", str(""), str(""))

	raw, _ := os.ReadFile(file)
	if string(raw) != "{}" {
		t.Fatalf("names.json = %s, want an empty object", raw)
	}
}

func TestImportFromLegacyDevices(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "devices.json")
	old := `{
  "192.168.1.9": {"ip": "192.168.1.9", "mac": "AA:BB:CC:DD:EE:01", "label": "Printer", "icon": "lucide:printer"},
  "192.168.1.10": {"ip": "192.168.1.10", "mac": "", "label": "NAS"},
  "192.168.1.11": {"ip": "192.168.1.11", "mac": "AA:BB:CC:DD:EE:03", "icon": "dashboard:plex"},
  "192.168.1.12": {"ip": "192.168.1.12", "mac": "AA:BB:CC:DD:EE:04", "hostname": "unnamed"}
}`
	if err := os.WriteFile(legacy, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	names := filepath.Join(dir, "names.json")

	n, err := OpenNames(names, legacy)
	if err != nil {
		t.Fatal(err)
	}

	if got := n.Get("aa:bb:cc:dd:ee:01"); got.Label != "Printer" || got.Icon != "lucide:printer" {
		t.Errorf("MAC-keyed import: %+v", got)
	}
	if got := n.Get("ip:192.168.1.10"); got.Label != "NAS" {
		t.Errorf("IP-keyed import: %+v", got)
	}
	if got := n.Get("aa:bb:cc:dd:ee:03"); got.Icon != "dashboard:plex" {
		t.Errorf("icon-only import: %+v", got)
	}
	if got := n.Get("aa:bb:cc:dd:ee:04"); got != (Name{}) {
		t.Errorf("a record with no label or icon must not be imported: %+v", got)
	}

	if _, err := os.Stat(names); err != nil {
		t.Fatalf("names.json was not written: %v", err)
	}
	after, _ := os.ReadFile(legacy)
	if string(after) != old {
		t.Error("devices.json must be left untouched")
	}
}

func TestImportRunsOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "devices.json")
	os.WriteFile(legacy, []byte(`{"10.0.0.2": {"ip": "10.0.0.2", "mac": "", "label": "Old"}}`), 0o600)
	names := filepath.Join(dir, "names.json")

	n, _ := OpenNames(names, legacy)
	n.Set("ip:10.0.0.2", str(""), nil) // user clears the imported name

	again, err := OpenNames(names, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Get("ip:10.0.0.2"); got != (Name{}) {
		t.Fatalf("names.json exists, so devices.json must not be imported again: %+v", got)
	}
}

func TestOpenNamesWithoutAnyFiles(t *testing.T) {
	dir := t.TempDir()
	n, err := OpenNames(filepath.Join(dir, "names.json"), filepath.Join(dir, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	if n.Get("anything") != (Name{}) {
		t.Fatal("expected no names")
	}
	if _, err := os.Stat(filepath.Join(dir, "names.json")); !os.IsNotExist(err) {
		t.Error("nothing to import, so no file should be written")
	}
}

func TestOpenNamesRejectsCorruptFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "names.json")
	os.WriteFile(file, []byte("not json"), 0o600)
	if _, err := OpenNames(file, ""); err == nil {
		t.Fatal("a corrupt names file should be reported, not silently discarded")
	}
}
