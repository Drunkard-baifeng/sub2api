//go:build unit

package admin

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsCanvas(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    map[string]any
		status  int
		enabled string
		url     string
	}{
		{"update", map[string]any{"canvas_enabled": true, "canvas_url": " https://example.com/new-canvas "}, http.StatusOK, "true", "https://example.com/new-canvas"},
		{"disable keeps address", map[string]any{"canvas_enabled": false}, http.StatusOK, "false", "https://example.com/canvas"},
		{"disabled clear", map[string]any{"canvas_enabled": false, "canvas_url": ""}, http.StatusOK, "false", ""},
		{"omitted", map[string]any{"site_name": "My Gateway"}, http.StatusOK, "true", "https://example.com/canvas"},
		{"enabled requires URL", map[string]any{"canvas_url": " "}, http.StatusBadRequest, "true", "https://example.com/canvas"},
		{"unsafe", map[string]any{"canvas_url": "javascript:alert(1)"}, http.StatusBadRequest, "true", "https://example.com/canvas"},
		{"disabled unsafe", map[string]any{"canvas_enabled": false, "canvas_url": "javascript:alert(1)"}, http.StatusBadRequest, "true", "https://example.com/canvas"},
		{"relative", map[string]any{"canvas_url": "/canvas"}, http.StatusBadRequest, "true", "https://example.com/canvas"},
		{"missing host", map[string]any{"canvas_url": "https://"}, http.StatusBadRequest, "true", "https://example.com/canvas"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{
				service.SettingKeyCanvasEnabled:     "true",
				service.SettingKeyCanvasURL:         "https://example.com/canvas",
				service.SettingKeyRedeemPurchaseURL: "https://example.com/shop",
			})
			rec := doUpdateSettings(t, h, tc.body, nil)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, tc.enabled, repo.values[service.SettingKeyCanvasEnabled])
			require.Equal(t, tc.url, repo.values[service.SettingKeyCanvasURL])
			require.Equal(t, "https://example.com/shop", repo.values[service.SettingKeyRedeemPurchaseURL])
			if tc.status == http.StatusOK {
				var response struct {
					Data struct {
						Enabled bool   `json:"canvas_enabled"`
						URL     string `json:"canvas_url"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
				require.Equal(t, tc.enabled == "true", response.Data.Enabled)
				require.Equal(t, tc.url, response.Data.URL)
			}
		})
	}
}
