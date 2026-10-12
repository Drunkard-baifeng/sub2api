package service

import "strings"

const DefaultCanvasURL = "https://image.jisuai.net"

// Existing installations get the default entry; an explicit false disables it.
func canvasEnabled(settings map[string]string) bool {
	value, ok := settings[SettingKeyCanvasEnabled]
	return !ok || value == "true"
}

func canvasURL(settings map[string]string) string {
	if value, ok := settings[SettingKeyCanvasURL]; ok {
		return strings.TrimSpace(value)
	}
	return DefaultCanvasURL
}
