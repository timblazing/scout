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

// Retired routes must not invoke host commands or disclose settings, even when
// requests satisfy the browser request-header check.
func TestRetiredRoutes(t *testing.T) {
	h := NewHandler(nil, config.Default())
	for _, path := range []string{"settings", "tailscale/connect", "tailscale/disconnect"} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			for _, suffix := range []string{"", "/"} {
				t.Run(method+"/"+path+suffix, func(t *testing.T) {
					req := httptest.NewRequest(method, "/api/"+path+suffix, nil)
					req.Header.Set("Content-Type", "application/json")
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, req)
					if rec.Code != http.StatusNotFound {
						t.Fatalf("status = %d: %s", rec.Code, rec.Body)
					}
					var body types.APIResponse
					if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if body.Success || body.Error != "endpoint not found" {
						t.Fatalf("unexpected response: %+v", body)
					}
				})
			}
		}
	}
}

func TestDeviceCustomizationEndpoint(t *testing.T) {
	dir := t.TempDir()
	devicesFile := filepath.Join(dir, "devices.json")
	stateFile := filepath.Join(dir, "state.json")
	store, err := storage.New(devicesFile, stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MergeDevices([]types.Device{{IP: "192.168.1.9"}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store, config.Default())
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"ip":"192.168.1.9","label":" Living room ","icon":"dashboard:raspberry-pi"}`, http.StatusOK},
		{`{"ip":"192.168.1.9","icon":"https://example.com/icon.svg"}`, http.StatusBadRequest},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/device", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("status = %d, want %d: %s", rec.Code, tc.want, rec.Body)
		}
	}
	reopened, err := storage.New(devicesFile, stateFile)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.GetDevice("192.168.1.9")
	if got.Label != "Living room" || got.Icon != "dashboard:raspberry-pi" {
		t.Fatalf("customization did not persist: %+v", got)
	}
}
