//go:build unit

package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsRedeemPurchaseURL(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   map[string]any
		status int
		want   string
	}{
		{"update", map[string]any{"redeem_purchase_url": " https://example.com/cdk "}, http.StatusOK, "https://example.com/cdk"},
		{"clear", map[string]any{"redeem_purchase_url": ""}, http.StatusOK, ""},
		{"omitted", map[string]any{"site_name": "My Gateway"}, http.StatusOK, "https://example.com/original"},
		{"unsafe", map[string]any{"redeem_purchase_url": "javascript:alert(1)"}, http.StatusBadRequest, "https://example.com/original"},
		{"relative", map[string]any{"redeem_purchase_url": "/shop"}, http.StatusBadRequest, "https://example.com/original"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{
				service.SettingKeyRedeemPurchaseURL:       "https://example.com/original",
				service.SettingKeyPurchaseSubscriptionURL: "https://example.com/subscriptions",
			})
			rec := doUpdateSettings(t, h, tc.body, nil)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, tc.want, repo.values[service.SettingKeyRedeemPurchaseURL])
			require.Equal(t, "https://example.com/subscriptions", repo.values[service.SettingKeyPurchaseSubscriptionURL])
		})
	}
}
