import type { PublicSettings } from '@/types'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

export const DEFAULT_CANVAS_URL = 'https://image.jisuai.net'

// Wait for settings so a disabled entry never flashes while the app loads.
export function resolveCanvasUrl(settings: Pick<PublicSettings, 'canvas_enabled' | 'canvas_url'> | null | undefined): string {
  if (!settings || !resolveFeatureFlag(settings, FeatureFlags.canvas)) return ''
  return sanitizeUrl(settings.canvas_url ?? DEFAULT_CANVAS_URL)
}
