package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/291-Group/LAN-Orangutan/internal/config"
	"github.com/291-Group/LAN-Orangutan/internal/storage"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

func TestOverview(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.New(filepath.Join(dir, "devices.json"), filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, d := range []types.Device{
		{IP: "192.168.1.20", LastSeen: now.Add(-2 * time.Hour), Risks: []string{"telnet"}},
		{IP: "192.168.1.3", LastSeen: now.Add(-time.Minute)},
		{IP: "192.168.1.100", LastSeen: now.Add(-30 * time.Minute), AddressHistory: []types.AddressChange{{IP: "192.168.1.9"}}},
	} {
		if err := store.UpdateDevice(&d); err != nil {
			t.Fatal(err)
		}
	}

	rec := httptest.NewRecorder()
	NewHandler(store, config.Default()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/overview", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}

	var resp struct {
		Data struct {
			Devices []struct {
				IP     string `json:"ip"`
				Status string `json:"status"`
			} `json:"devices"`
			Flagged int `json:"flagged"`
			Moved   int `json:"moved"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	got := resp.Data.Devices
	want := []struct{ ip, status string }{
		{"192.168.1.3", "online"},
		{"192.168.1.20", "offline"},
		{"192.168.1.100", "seen"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d devices, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].IP != w.ip || got[i].Status != w.status {
			t.Errorf("device %d = %s/%s, want %s/%s", i, got[i].IP, got[i].Status, w.ip, w.status)
		}
	}
	if resp.Data.Flagged != 1 || resp.Data.Moved != 1 {
		t.Errorf("flagged/moved = %d/%d, want 1/1", resp.Data.Flagged, resp.Data.Moved)
	}
}
