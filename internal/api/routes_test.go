package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/291-Group/LAN-Orangutan/internal/config"
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
