package service

import "strings"

const DefaultRedeemPurchaseURL = "https://catfk.com/shop/7LOVBPOL"

// An absent setting uses the site's default shop. An explicit empty value hides
// the purchase entry, so it must not use the usual empty-string fallback.
func redeemPurchaseURL(settings map[string]string) string {
	if value, ok := settings[SettingKeyRedeemPurchaseURL]; ok {
		return strings.TrimSpace(value)
	}
	return DefaultRedeemPurchaseURL
}
