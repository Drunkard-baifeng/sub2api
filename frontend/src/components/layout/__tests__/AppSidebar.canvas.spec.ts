import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppSidebar from '../AppSidebar.vue'
import { DEFAULT_CANVAS_URL, resolveCanvasUrl } from '@/config/canvas'

const app = reactive({
  cachedPublicSettings: { canvas_enabled: true, canvas_url: DEFAULT_CANVAS_URL, custom_menu_items: [] },
  sidebarCollapsed: false, mobileOpen: false, backendModeEnabled: false, sidebarScrollTop: 0,
  publicSettingsLoaded: true, siteName: '极速 AI', siteLogo: '', siteVersion: 'test',
  setMobileOpen: vi.fn((value: boolean) => { app.mobileOpen = value }),
  toggleSidebar: vi.fn(),
})
const auth = reactive({ isAdmin: false, isSimpleMode: false, token: 'private-token', user: { id: 123 } })
const admin = { customMenuItems: [], fetch: vi.fn().mockResolvedValue(undefined) }

vi.mock('@/stores', () => ({
  useAppStore: () => app,
  useAuthStore: () => auth,
  useAdminSettingsStore: () => admin,
  useOnboardingStore: () => ({ isCurrentStep: () => false, nextStep: vi.fn() }),
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/components/common/VersionBadge.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage: ref(false), refreshBatchImageAccess: vi.fn() }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const wrappers: VueWrapper[] = []
async function mountSidebar() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  await router.push('/dashboard')
  const wrapper = mount(AppSidebar, { global: { plugins: [router], stubs: { VersionBadge: true } } })
  wrappers.push(wrapper)
  await flushPromises()
  return { wrapper, router }
}
beforeEach(() => {
  app.cachedPublicSettings = { canvas_enabled: true, canvas_url: DEFAULT_CANVAS_URL, custom_menu_items: [] }
  app.sidebarCollapsed = false
  app.mobileOpen = false
  app.backendModeEnabled = false
  auth.isAdmin = false
  auth.isSimpleMode = false
  vi.clearAllMocks()
})
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

describe('Canvas URL resolution', () => {
  it('waits for settings, supports older backends, and respects disabled or empty entries', () => {
    expect(resolveCanvasUrl(null)).toBe('')
    expect(resolveCanvasUrl({})).toBe(DEFAULT_CANVAS_URL + '/')
    expect(resolveCanvasUrl({ canvas_enabled: false, canvas_url: DEFAULT_CANVAS_URL })).toBe('')
    expect(resolveCanvasUrl({ canvas_enabled: true, canvas_url: '' })).toBe('')
    expect(resolveCanvasUrl({ canvas_enabled: true })).toBe(DEFAULT_CANVAS_URL + '/')
  })
  it.each(['javascript:alert(1)', 'data:text/html,test', '/canvas', '//example.com', 'https://'])('rejects %s', url => {
    expect(resolveCanvasUrl({ canvas_enabled: true, canvas_url: url })).toBe('')
  })
})

describe('Canvas sidebar entry', () => {
  it.each([false, true])('renders once for administrator=%s as a plain external anchor', async isAdmin => {
    auth.isAdmin = isAdmin
    const { wrapper, router } = await mountSidebar()
    const links = wrapper.findAll('a[target="_blank"]')
    expect(links).toHaveLength(1)
    const link = links[0]
    const tools = wrapper.get('[data-nav-section="tools"]')
    expect(tools.get('h3').text()).toBe('nav.sectionTools')
    expect(tools.get('a[target="_blank"]').element).toBe(link.element)
    expect(wrapper.get('[data-nav-section="other"]').find('a[target="_blank"]').exists()).toBe(false)
    expect(wrapper.get('[data-nav-section="other"] a[href="/profile"]').exists()).toBe(true)
    expect(link.attributes('href')).toBe(DEFAULT_CANVAS_URL + '/')
    expect(link.attributes('rel')).toBe('noopener noreferrer')
    expect(link.attributes('aria-label')).toBe('nav.canvasNewTab')
    expect(link.classes()).not.toContain('sidebar-link-active')
    expect(link.find('svg').exists()).toBe(true)
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(link.attributes('href')).not.toMatch(/token|api_key|user_id/)
    await link.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/dashboard')
    // Existing internal links retain RouterLink navigation.
    await wrapper.get('a[href="/keys"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/keys')
  })
  it('reacts to changed settings without a reload', async () => {
    const { wrapper } = await mountSidebar()
    app.cachedPublicSettings.canvas_enabled = false
    await flushPromises()
    expect(wrapper.find('a[target="_blank"]').exists()).toBe(false)
    expect(wrapper.find('[data-nav-section="tools"]').exists()).toBe(false)
    app.cachedPublicSettings.canvas_url = ' https://example.com/canvas?mode=draw '
    app.cachedPublicSettings.canvas_enabled = true
    await flushPromises()
    expect(wrapper.get('a[target="_blank"]').attributes('href')).toBe('https://example.com/canvas?mode=draw')
    app.cachedPublicSettings.canvas_url = 'javascript:alert(1)'
    await flushPromises()
    expect(wrapper.find('a[target="_blank"]').exists()).toBe(false)
  })
  it('closes the mobile drawer and keeps an accessible label when collapsed', async () => {
    app.mobileOpen = true
    app.sidebarCollapsed = true
    const { wrapper } = await mountSidebar()
    const link = wrapper.get('a[target="_blank"]')
    expect(link.classes()).toContain('sidebar-link-collapsed')
    expect(link.attributes('title')).toBe('nav.canvasNewTab')
    await link.trigger('click')
    expect(app.setMobileOpen).toHaveBeenCalledWith(false)
    expect(app.mobileOpen).toBe(false)
  })
})
