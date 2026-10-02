//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRedeemPurchaseURLSettings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"default", map[string]string{}, DefaultRedeemPurchaseURL},
		{"configured", map[string]string{SettingKeyRedeemPurchaseURL: " https://example.com/cdk "}, "https://example.com/cdk"},
		{"hidden", map[string]string{SettingKeyRedeemPurchaseURL: ""}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: tc.values}, &config.Config{})
			public, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.want, public.RedeemPurchaseURL)
			require.Equal(t, tc.want, svc.parseSettings(tc.values).RedeemPurchaseURL)
			injected, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.want, injected.(*PublicSettingsInjectionPayload).RedeemPurchaseURL)
		})
	}
}
