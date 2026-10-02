import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { nextTick } from 'vue'
import HeaderWallet from '../HeaderWallet.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const wrappers: VueWrapper[] = []
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  vi.useRealTimers()
})

async function mountWallet(overrides = {}) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/dashboard', '/purchase', '/redeem', '/orders'].map(path => ({ path, component: { template: '<div />' } })),
  })
  await router.push('/dashboard')
  const wrapper = mount(HeaderWallet, {
    attachTo: document.body,
    props: {
      open: false,
      balance: 10.25,
      frozenBalance: 2.5,
      rechargeEnabled: true,
      redeemEnabled: true,
      ordersEnabled: true,
      ...overrides,
      'onUpdate:open': (open: boolean) => { void wrapper.setProps({ open }) },
    },
    global: { plugins: [router], stubs: { transition: true } },
  })
  wrappers.push(wrapper)
  return { wrapper, router }
}

describe('HeaderWallet', () => {
  it('shows distinct available, frozen and total amounts and reacts to balance updates', async () => {
    const { wrapper } = await mountWallet()
    await wrapper.get('.wallet-trigger').trigger('click')
    expect(wrapper.get('[data-testid="wallet-available"]').text()).toBe('$10.25')
    expect(wrapper.get('[data-testid="wallet-frozen"]').text()).toBe('$2.50')
    expect(wrapper.get('[data-testid="wallet-total"]').text()).toBe('$12.75')
    await wrapper.setProps({ balance: -1.25, frozenBalance: 3 })
    expect(wrapper.get('[data-testid="wallet-available"]').text()).toBe('-$1.25')
    expect(wrapper.get('[data-testid="wallet-total"]').text()).toBe('$1.75')
  })

  it('keeps the hover panel open while crossing to its actions and closes on mouse leave', async () => {
    vi.useFakeTimers()
    const { wrapper } = await mountWallet()
    await wrapper.trigger('pointerenter', { pointerType: 'mouse' })
    expect(wrapper.get('.wallet-trigger').attributes('aria-expanded')).toBe('true')
    await wrapper.trigger('pointerleave', { pointerType: 'mouse' })
    vi.advanceTimersByTime(100)
    await wrapper.trigger('pointerenter', { pointerType: 'mouse' })
    vi.advanceTimersByTime(200)
    await nextTick()
    expect(wrapper.find('.wallet-panel').exists()).toBe(true)
    await wrapper.trigger('pointerleave', { pointerType: 'mouse' })
    vi.advanceTimersByTime(180)
    await nextTick()
    expect(wrapper.find('.wallet-panel').exists()).toBe(false)
  })

  it('lets a click pin a hovered panel and a second click close it', async () => {
    const { wrapper } = await mountWallet()
    await wrapper.trigger('pointerenter', { pointerType: 'mouse' })
    const button = wrapper.get('.wallet-trigger')
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('true')
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('false')
  })

  it('supports touch clicks without treating touch as hover', async () => {
    const { wrapper } = await mountWallet()
    await wrapper.trigger('pointerenter', { pointerType: 'touch' })
    expect(wrapper.find('.wallet-panel').exists()).toBe(false)
    await wrapper.get('.wallet-trigger').trigger('click')
    expect(wrapper.find('.wallet-panel').exists()).toBe(true)
  })

  it('closes on Escape with focus restored, and on outside clicks', async () => {
    const { wrapper } = await mountWallet()
    await wrapper.get('.wallet-trigger').trigger('click')
    const recharge = wrapper.get('.wallet-recharge').element as HTMLAnchorElement
    recharge.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.find('.wallet-panel').exists()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('.wallet-trigger').element)
    await wrapper.get('.wallet-trigger').trigger('click')
    document.body.click()
    await nextTick()
    expect(wrapper.find('.wallet-panel').exists()).toBe(false)
  })

  it.each(['/purchase', '/redeem', '/orders'])('navigates to %s and closes the panel', async path => {
    const { wrapper, router } = await mountWallet()
    await wrapper.get('.wallet-trigger').trigger('click')
    await wrapper.get(`.wallet-panel a[href="${path}"]`).trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe(path)
    expect(wrapper.find('.wallet-panel').exists()).toBe(false)
  })

  it('hides disabled recharge and order entries while leaving redemption available', async () => {
    const { wrapper } = await mountWallet({ rechargeEnabled: false, ordersEnabled: false })
    await wrapper.get('.wallet-trigger').trigger('click')
    expect(wrapper.find('a[href="/purchase"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/orders"]').exists()).toBe(false)
    expect(wrapper.get('a[href="/redeem"]').exists()).toBe(true)
    await wrapper.setProps({ redeemEnabled: false })
    expect(wrapper.find('.wallet-actions').exists()).toBe(false)
    expect(wrapper.get('[data-testid="wallet-total"]').text()).toBe('$12.75')
  })
})
