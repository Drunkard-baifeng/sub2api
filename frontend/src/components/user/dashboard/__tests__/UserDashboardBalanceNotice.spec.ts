import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, nextTick, reactive } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { PublicSettings } from '@/types'
import UserDashboardBalanceNotice from '../UserDashboardBalanceNotice.vue'

const { useAppStoreMock } = vi.hoisted(() => ({ useAppStoreMock: vi.fn() }))
vi.mock('@/stores/app', () => ({ useAppStore: useAppStoreMock }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

let settings: {
  publicSettingsLoaded: boolean
  cachedPublicSettings: Partial<PublicSettings>
}

beforeEach(() => {
  settings = reactive({
    publicSettingsLoaded: true,
    cachedPublicSettings: { payment_enabled: true, payment_balance_disabled: false },
  })
  useAppStoreMock.mockReturnValue(settings)
})

async function mountNotice(balance = 0, isSimple = false) {
  const page = defineComponent({ template: '<div />' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: page },
      { path: '/purchase', component: page },
    ],
  })
  await router.push('/dashboard')
  const wrapper = mount(UserDashboardBalanceNotice, {
    props: { balance, isSimple },
    global: {
      plugins: [router],
    },
  })
  return { wrapper, router }
}

describe('UserDashboardBalanceNotice', () => {
  it('shows the actual zero balance and navigates to the recharge page', async () => {
    const { wrapper, router } = await mountNotice()
    expect(wrapper.get('[role="status"]').text()).toContain('dashboard.balanceNotice.title')
    expect(wrapper.text()).toContain('$0.00')
    await wrapper.get('a').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/purchase')
  })

  it('keeps the recharge link while the refreshed balance switches between warning and normal states', async () => {
    const { wrapper } = await mountNotice(-1.25)
    expect(wrapper.text()).toContain('-$1.25')
    expect(wrapper.get('h2').text()).toBe('dashboard.balanceNotice.title')
    await wrapper.setProps({ balance: 10 })
    expect(wrapper.get('h2').text()).toBe('dashboard.balanceNotice.balanceTitle')
    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.get('a').attributes('href')).toBe('/purchase')
    await wrapper.setProps({ balance: 0 })
    expect(wrapper.get('h2').text()).toBe('dashboard.balanceNotice.title')
    expect(wrapper.get('a').attributes('href')).toBe('/purchase')
  })

  it.each([0.001, 1, 10])('keeps the recharge entry visible for a positive balance (%s)', async balance => {
    const { wrapper, router } = await mountNotice(balance)
    expect(wrapper.get('h2').text()).toBe('dashboard.balanceNotice.balanceTitle')
    expect(wrapper.get('[role="status"]').text()).toContain('dashboard.balanceNotice.balanceDescription')
    await wrapper.get('a').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/purchase')
  })

  it.each([NaN, Infinity, -Infinity])('does not show an invalid balance (%s)', async balance => {
    const { wrapper } = await mountNotice(balance)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('waits for public settings before showing a recharge link', async () => {
    settings.publicSettingsLoaded = false
    const { wrapper } = await mountNotice()
    expect(wrapper.find('a').exists()).toBe(false)
    settings.publicSettingsLoaded = true
    await nextTick()
    expect(wrapper.find('a').exists()).toBe(true)
  })

  it.each(['payment', 'balance'])('hides the notice when %s payments are disabled', async disabled => {
    if (disabled === 'payment') settings.cachedPublicSettings.payment_enabled = false
    else settings.cachedPublicSettings.payment_balance_disabled = true
    const { wrapper } = await mountNotice()
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('hides the notice in simple mode', async () => {
    const { wrapper } = await mountNotice(0, true)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })
})
