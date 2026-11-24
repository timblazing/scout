package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/291-Group/LAN-Orangutan/internal/config"
	"github.com/291-Group/LAN-Orangutan/internal/storage"
	"github.com/291-Group/LAN-Orangutan/internal/types"
)

func newTestHandler(t *testing.T, devices ...types.Device) (*Handler, string) {
	t.Helper()
	namesFile := filepath.Join(t.TempDir(), "names.json")
	names, err := storage.OpenNames(namesFile, "")
	if err != nil {
		t.Fatal(err)
	}
	live := storage.NewLive()
	live.Apply(devices, nil, nil)
	return NewHandler(live, names, config.Default()), namesFile
}

func do(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type listResponse struct {
	Data struct {
		Devices []struct {
			ID    string `json:"id"`
			IP    string `json:"ip"`
			MAC   string `json:"mac"`
			Label string `json:"label"`
			Icon  string `json:"icon"`
			Type  string `json:"type"`
		} `json:"devices"`
		Scanning bool `json:"scanning"`
	} `json:"data"`
}

func TestListDevicesSortedAndNamed(t *testing.T) {
	h, _ := newTestHandler(t,
		types.Device{IP: "192.168.1.100", MAC: "AA:BB:CC:00:00:02"},
		types.Device{IP: "192.168.1.20"},
		types.Device{IP: "192.168.1.3", MAC: "AA:BB:CC:00:00:01"},
	)
	if err := h.names.Set("aa:bb:cc:00:00:01", ptr("Router"), ptr("lucide:router")); err != nil {
		t.Fatal(err)
	}

	rec := do(h, http.MethodGet, "/api/devices", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var resp listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	got := resp.Data.Devices
	wantIPs := []string{"192.168.1.3", "192.168.1.20", "192.168.1.100"}
	if len(got) != len(wantIPs) {
		t.Fatalf("got %d devices, want %d", len(got), len(wantIPs))
	}
	for i, ip := range wantIPs {
		if got[i].IP != ip {
			t.Errorf("device %d = %s, want %s", i, got[i].IP, ip)
		}
	}
	if got[0].ID != "aa:bb:cc:00:00:01" || got[0].Label != "Router" || got[0].Icon != "lucide:router" {
		t.Errorf("names must be applied by key: %+v", got[0])
	}
	if got[1].ID != "ip:192.168.1.20" {
		t.Errorf("device without a MAC is keyed by IP: %+v", got[1])
	}
	if resp.Data.Scanning {
		t.Error("scanning must be false once a scan cycle has completed")
	}
}

func TestListDevicesBeforeFirstScan(t *testing.T) {
	names, _ := storage.OpenNames(filepath.Join(t.TempDir(), "names.json"), "")
	h := NewHandler(storage.NewLive(), names, config.Default())

	var resp listResponse
	rec := do(h, http.MethodGet, "/api/devices", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Data.Scanning || len(resp.Data.Devices) != 0 {
		t.Fatalf("want an empty, scanning list: %s", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"devices":[]`) {
		t.Errorf("devices must be an empty array, not null: %s", rec.Body)
	}
}

func TestPatchDevicePersistsLabel(t *testing.T) {
	h, namesFile := newTestHandler(t, types.Device{IP: "192.168.1.9", MAC: "AA:BB:CC:00:00:09"})

	rec := do(h, http.MethodPatch, "/api/devices/aa:bb:cc:00:00:09", `{"label":" Living room ","icon":"dashboard:raspberry-pi"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}

	reopened, err := storage.OpenNames(namesFile, "")
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get("aa:bb:cc:00:00:09")
	if got.Label != "Living room" || got.Icon != "dashboard:raspberry-pi" {
		t.Fatalf("customization did not persist: %+v", got)
	}

	// An id that is not a MAC (IP-keyed) works too, URL-encoded as the UI sends it.
	h2, _ := newTestHandler(t, types.Device{IP: "192.168.1.20"})
	if rec := do(h2, http.MethodPatch, "/api/devices/ip%3A192.168.1.20", `{"label":"NAS"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if got := h2.names.Get("ip:192.168.1.20"); got.Label != "NAS" {
		t.Fatalf("%+v", got)
	}
}

func TestPatchDeviceRejections(t *testing.T) {
	h, _ := newTestHandler(t, types.Device{IP: "192.168.1.9", MAC: "aa:bb:cc:00:00:09"})
	for _, tc := range []struct {
		name, path, body string
		want             int
	}{
		{"unknown id", "/api/devices/aa:bb:cc:00:00:99", `{"label":"x"}`, http.StatusNotFound},
		{"bad icon", "/api/devices/aa:bb:cc:00:00:09", `{"icon":"https://example.com/icon.svg"}`, http.StatusBadRequest},
		{"long name", "/api/devices/aa:bb:cc:00:00:09", `{"label":"` + strings.Repeat("x", 121) + `"}`, http.StatusBadRequest},
		{"bad json", "/api/devices/aa:bb:cc:00:00:09", `{`, http.StatusBadRequest},
	} {
		if rec := do(h, http.MethodPatch, tc.path, tc.body); rec.Code != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", tc.name, rec.Code, tc.want, rec.Body)
		}
	}
	if got := h.names.Get("aa:bb:cc:00:00:99"); got != (storage.Name{}) {
		t.Error("an unknown device must not get a saved name")
	}
}

func TestMethodsAndCSRF(t *testing.T) {
	h, _ := newTestHandler(t, types.Device{IP: "192.168.1.9", MAC: "aa:bb:cc:00:00:09"})

	if rec := do(h, http.MethodPost, "/api/devices", `{}`); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/devices = %d, want 405", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/api/devices/aa:bb:cc:00:00:09", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/devices/{id} = %d, want 405", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/devices/aa:bb:cc:00:00:09", strings.NewReader(`{"label":"x"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("PATCH without a JSON content type = %d, want 403", rec.Code)
	}
}

// Every endpoint from before the rework, and anything else, is gone.
func TestRemovedRoutes(t *testing.T) {
	h, _ := newTestHandler(t)
	for _, path := range []string{
		"device", "networks", "scan", "scan/progress", "scan/cancel",
		"scan/continuous", "tailscale", "stats", "status", "settings", "tailscale/connect",
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			for _, suffix := range []string{"", "/"} {
				rec := do(h, method, "/api/"+path+suffix, "")
				if rec.Code != http.StatusNotFound {
					t.Errorf("%s /api/%s%s = %d: %s", method, path, suffix, rec.Code, rec.Body)
					continue
				}
				var body types.APIResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Success || body.Error != "endpoint not found" {
					t.Errorf("%s /api/%s%s: unexpected response %s", method, path, suffix, rec.Body)
				}
			}
		}
	}
}

func ptr(s string) *string { return &s }
