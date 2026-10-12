//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCanvasSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		values  map[string]string
		enabled bool
		url     string
	}{
		{"defaults", map[string]string{}, true, DefaultCanvasURL},
		{"configured", map[string]string{SettingKeyCanvasEnabled: "true", SettingKeyCanvasURL: " https://example.com/canvas "}, true, "https://example.com/canvas"},
		{"disabled keeps URL", map[string]string{SettingKeyCanvasEnabled: "false", SettingKeyCanvasURL: "https://example.com/canvas"}, false, "https://example.com/canvas"},
		{"disabled and cleared", map[string]string{SettingKeyCanvasEnabled: "false", SettingKeyCanvasURL: ""}, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: tc.values}, &config.Config{})
			public, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.enabled, public.CanvasEnabled)
			require.Equal(t, tc.url, public.CanvasURL)
			admin := svc.parseSettings(tc.values)
			require.Equal(t, tc.enabled, admin.CanvasEnabled)
			require.Equal(t, tc.url, admin.CanvasURL)
			injected, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.enabled, injected.(*PublicSettingsInjectionPayload).CanvasEnabled)
			require.Equal(t, tc.url, injected.(*PublicSettingsInjectionPayload).CanvasURL)
		})
	}
}
