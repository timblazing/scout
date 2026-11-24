// Package storage holds Scout's device state: the live set of devices currently
// on the network (memory only) and the names and icons the user has chosen
// (the one thing written to disk).
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Name is what the user has chosen for a device.
type Name struct {
	Label string `json:"label,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

// Names persists user-chosen labels and icons, keyed by device identity (see
// Key). It is the only state Scout writes to disk.
type Names struct {
	file string
	mu   sync.RWMutex
	data map[string]Name
}

// OpenNames loads the names file. When it does not exist yet and a
// legacyDevicesFile from an older version does, the labels and icons in it are
// imported once and written to the names file; the old file is left untouched.
// legacyDevicesFile may be empty.
func OpenNames(file, legacyDevicesFile string) (*Names, error) {
	n := &Names{file: file, data: make(map[string]Name)}

	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	raw, err := os.ReadFile(file)
	switch {
	case err == nil:
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &n.data); err != nil {
				return nil, fmt.Errorf("could not read the device names at %s: %w", file, err)
			}
		}
		if n.data == nil {
			n.data = make(map[string]Name)
		}
	case os.IsNotExist(err):
		if legacyDevicesFile != "" {
			imported, err := importLegacy(legacyDevicesFile)
			if err != nil {
				return nil, err
			}
			if len(imported) > 0 {
				n.data = imported
				if err := n.save(); err != nil {
					return nil, err
				}
			}
		}
	default:
		return nil, fmt.Errorf("could not read the device names at %s: %w", file, err)
	}
	return n, nil
}

// importLegacy reads names and icons from a devices.json written by an older
// version, which keyed records by IP. A missing file imports nothing.
func importLegacy(file string) (map[string]Name, error) {
	raw, err := os.ReadFile(file)
	if os.IsNotExist(err) || (err == nil && len(raw) == 0) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read the device list at %s: %w", file, err)
	}

	var old map[string]struct {
		IP    string `json:"ip"`
		MAC   string `json:"mac"`
		Label string `json:"label"`
		Icon  string `json:"icon"`
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		return nil, fmt.Errorf("could not read the device list at %s: %w", file, err)
	}

	out := make(map[string]Name)
	for ip, rec := range old {
		label := strings.TrimSpace(rec.Label)
		if label == "" && rec.Icon == "" {
			continue
		}
		if rec.IP != "" {
			ip = rec.IP
		}
		out[Key(rec.MAC, ip)] = Name{Label: label, Icon: rec.Icon}
	}
	return out, nil
}

// Get returns the name saved for a device key, or the zero value.
func (n *Names) Get(key string) Name {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.data[key]
}

// Set updates the label and/or icon saved for a device key. A nil field is left
// as it was; an empty string clears it. An entry with neither is removed.
func (n *Names) Set(key string, label, icon *string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	previous, had := n.data[key]
	next := previous
	if label != nil {
		next.Label = *label
	}
	if icon != nil {
		next.Icon = *icon
	}
	if next == (Name{}) {
		delete(n.data, key)
	} else {
		n.data[key] = next
	}

	if err := n.save(); err != nil {
		if had {
			n.data[key] = previous
		} else {
			delete(n.data, key)
		}
		return err
	}
	return nil
}

// save writes the names file atomically. Callers hold n.mu, or have not shared n.
func (n *Names) save() error {
	raw, err := json.MarshalIndent(n.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal names: %w", err)
	}
	return atomicWrite(n.file, raw)
}

// atomicWrite writes data to a file atomically using a temp file, owner-only.
func atomicWrite(path string, data []byte) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		if tempPath != "" {
			os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	tempPath = "" // renamed; nothing to clean up
	return nil
}
