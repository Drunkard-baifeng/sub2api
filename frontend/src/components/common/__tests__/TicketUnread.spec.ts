import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import TicketUnread from '../TicketUnread.vue'

const mocks = vi.hoisted(() => ({ unread: vi.fn() }))
const auth = reactive<{ user: { id: number; role: string } | null }>({ user: { id: 1, role: 'user' } })
const route = reactive({ path: '/dashboard' })
vi.mock('@/api/tickets', () => ({ ticketsAPI: mocks }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('vue-router', () => ({ useRoute: () => route }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
function setup() {
  return mount(TicketUnread, { global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } } })
}
beforeEach(() => {
  vi.resetAllMocks()
  auth.user = { id: 1, role: 'user' }; route.path = '/dashboard'
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  mocks.unread.mockResolvedValue(3)
})
describe('Ticket unread badge', () => {
  it('refreshes on changes and separates admin and personal unread scopes', async () => {
    auth.user = { id: 1, role: 'admin' }; route.path = '/admin/dashboard'
    const wrapper = setup(); await flushPromises()
    expect(mocks.unread).toHaveBeenCalledWith(true)
    expect(wrapper.get('a').attributes('href')).toBe('/admin/tickets')
    expect(wrapper.text()).toBe('3')
    route.path = '/dashboard'; await flushPromises()
    expect(mocks.unread).toHaveBeenLastCalledWith(false)
    expect(wrapper.get('a').attributes('href')).toBe('/tickets')
    mocks.unread.mockResolvedValue(1); window.dispatchEvent(new Event('tickets-changed')); await flushPromises()
    expect(wrapper.text()).toBe('1')
  })
  it('ignores a response from an account that has logged out', async () => {
    let finish!: (value: number) => void
    mocks.unread.mockReturnValue(new Promise(resolve => { finish = resolve }))
    const wrapper = setup()
    auth.user = null; await flushPromises(); finish(7); await flushPromises()
    expect(wrapper.text()).toBe('')
  })
  it('stops polling when hidden and removes listeners on unmount', async () => {
    const wrapper = setup(); await flushPromises()
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    window.dispatchEvent(new Event('tickets-changed')); await flushPromises()
    expect(mocks.unread).toHaveBeenCalledOnce()
    wrapper.unmount()
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
    window.dispatchEvent(new Event('tickets-changed')); await flushPromises()
    expect(mocks.unread).toHaveBeenCalledOnce()
  })
})
