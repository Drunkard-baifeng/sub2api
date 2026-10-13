import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import TicketsView from '../TicketsView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(), detail: vi.fn(), create: vi.fn(), reply: vi.fn(), read: vi.fn(), status: vi.fn(), attachment: vi.fn(), remove: vi.fn(),
}))
const route = reactive({ path: '/tickets' })
const auth = reactive({ user: { id: 1, role: 'user' } })
vi.mock('@/api/tickets', () => ({ ticketsAPI: mocks, notifyTicketsChanged: vi.fn() }))
vi.mock('vue-router', () => ({ useRoute: () => route }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }) }))
enableAutoUnmount(afterEach)

const ticket = (id: number, status = 'pending') => ({
  id, user_id: 1, subject: 'Problem ' + id, category: 'api', status, unread: true,
  last_message_id: id * 10, created_at: '2026-10-12T10:00:00Z', updated_at: '2026-10-12T10:00:00Z',
})
const detail = (id: number, status = 'pending') => ({
  ticket: ticket(id, status), has_more: false,
  messages: [{ id: id * 10, is_admin: false, body: 'Message ' + id, status_event: '', created_at: '2026-10-12T10:00:00Z', attachments: [] }],
})
function setup() {
  return mount(TicketsView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, TicketImage: true, ConfirmDialog: { props: ['show', 'message'], emits: ['confirm', 'cancel'], template: '<div v-if="show" data-test="confirm"><p>{{ message }}</p><button data-test="confirm-delete" @click="$emit(\'confirm\')">Confirm</button><button data-test="cancel-delete" @click="$emit(\'cancel\')">Cancel</button></div>' } } } })
}
beforeEach(() => {
  vi.resetAllMocks()
  route.path = '/tickets'; auth.user = { id: 1, role: 'user' }
  mocks.list.mockResolvedValue({ items: [ticket(1), ticket(2)], total: 2 })
  mocks.detail.mockImplementation((_admin, id) => Promise.resolve(detail(id)))
  mocks.read.mockResolvedValue(undefined)
  mocks.create.mockResolvedValue(3)
  mocks.reply.mockResolvedValue(undefined)
  mocks.status.mockResolvedValue(undefined)
  mocks.remove.mockResolvedValue(undefined)
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:preview'), revokeObjectURL: vi.fn() }))
})
afterEach(() => vi.unstubAllGlobals())
describe('Ticket center', () => {
  it('shows retention and offers no user deletion even for resolved tickets', async () => {
    mocks.detail.mockResolvedValue(detail(1, 'resolved'))
    const wrapper = setup(); await flushPromises()
    expect(wrapper.text()).toContain('tickets.retentionNotice')
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    expect(wrapper.findAll('button').some(b => b.text() === 'tickets.delete')).toBe(false)
    expect(wrapper.find('[data-test="confirm"]').exists()).toBe(false)
    expect(mocks.remove).not.toHaveBeenCalled()
  })
  it('requires admin confirmation and clears the deleted conversation', async () => {
    route.path = '/admin/tickets'; auth.user.role = 'admin'
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.delete')!.trigger('click')
    expect(mocks.remove).not.toHaveBeenCalled()
    await wrapper.get('[data-test="cancel-delete"]').trigger('click')
    expect(mocks.remove).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.delete')!.trigger('click')
    mocks.list.mockResolvedValue({ items: [ticket(2)], total: 1 })
    await wrapper.get('[data-test="confirm-delete"]').trigger('click'); await flushPromises()
    expect(mocks.remove).toHaveBeenCalledTimes(1)
    expect(mocks.remove).toHaveBeenCalledWith(1)
    expect(wrapper.text()).not.toContain('Message 1')
    expect(wrapper.text()).not.toContain('Problem 1')
    expect(wrapper.text()).toContain('Problem 2')
  })
  it('preserves the conversation and draft when admin deletion fails', async () => {
    route.path = '/admin/tickets'; auth.user.role = 'admin'
    mocks.remove.mockRejectedValue({ status: 503, message: 'Try later' })
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    await wrapper.get('textarea').setValue('Keep this draft')
    await wrapper.findAll('button').find(b => b.text() === 'tickets.delete')!.trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click'); await flushPromises()
    expect(wrapper.text()).toContain('Message 1')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Keep this draft')
    expect(wrapper.get('[role="alert"]').text()).toContain('Try later')
  })
  it('clears an expired conversation after a refresh returns 404', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    mocks.detail.mockRejectedValue({ status: 404 })
    mocks.list.mockResolvedValue({ items: [], total: 0 })
    await wrapper.get('button[aria-label="tickets.refresh"]').trigger('click'); await flushPromises()
    expect(wrapper.text()).not.toContain('Message 1')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toBe('tickets.expiredOrDeleted')
  })
  it('drops a pending deletion confirmation when switching account', async () => {
    route.path = '/admin/tickets'; auth.user.role = 'admin'
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.delete')!.trigger('click')
    auth.user = { id: 2, role: 'admin' }; await flushPromises()
    expect(wrapper.find('[data-test="confirm"]').exists()).toBe(false)
    expect(mocks.remove).not.toHaveBeenCalled()
  })
  it('uses a compact list heading instead of duplicating the page title', async () => {
    const wrapper = setup(); await flushPromises()
    expect(wrapper.get('h2').text()).toBe('tickets.myTickets')
    expect(wrapper.text()).not.toContain('tickets.subtitle')
    route.path = '/admin/tickets'; await flushPromises()
    expect(wrapper.get('h2').text()).toBe('tickets.allTickets')
    expect(wrapper.text()).not.toContain('tickets.adminSubtitle')
  })
  it('accepts screenshot clipboard items from the whole composer without duplicate files', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.create')!.trigger('click')
    const screenshot = new File(['png'], 'clipboard.png', { type: 'image/png' })
    const getAsFile = vi.fn(() => screenshot)
    await wrapper.get('form').trigger('paste', { clipboardData: { files: [], items: [{ kind: 'file', type: 'image/png', getAsFile }] } })
    expect(wrapper.findAll('img')).toHaveLength(1)
    expect(getAsFile).toHaveBeenCalledTimes(1)
    await wrapper.get('form').trigger('paste', { clipboardData: { files: [screenshot], items: [{ kind: 'file', type: 'image/png', getAsFile }] } })
    expect(wrapper.findAll('img')).toHaveLength(2)
    expect(getAsFile).toHaveBeenCalledTimes(1)
  })
  it('leaves normal text paste to the browser', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.create')!.trigger('click')
    const pasteEvent = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(pasteEvent, 'clipboardData', { value: { files: [], items: [{ kind: 'string', type: 'text/plain' }] } })
    wrapper.get('textarea').element.dispatchEvent(pasteEvent)
    expect(pasteEvent.defaultPrevented).toBe(false)
    expect(wrapper.find('img').exists()).toBe(false)
  })
  it('preserves the draft when the same account profile refreshes', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.create')!.trigger('click')
    await wrapper.get('input[maxlength="120"]').setValue('Draft subject')
    await wrapper.get('textarea').setValue('Keep my draft')
    const calls = mocks.list.mock.calls.length
    auth.user = { id: 1, role: 'user' }
    await flushPromises()
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Keep my draft')
    expect((wrapper.get('input[maxlength="120"]').element as HTMLInputElement).value).toBe('Draft subject')
    expect(mocks.list).toHaveBeenCalledTimes(calls)
  })
  it('keeps the new-ticket action disabled during drafting and offers cancel', async () => {
    const wrapper = setup(); await flushPromises()
    const create = wrapper.findAll('button').find(b => b.text() === 'tickets.create')!
    await create.trigger('click')
    await wrapper.get('input[maxlength="120"]').setValue('Draft subject')
    await wrapper.get('textarea').setValue('Draft message')
    expect((create.element as HTMLButtonElement).disabled).toBe(true)
    await wrapper.findAll('button').find(b => b.text() === 'tickets.cancel')!.trigger('click')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect((create.element as HTMLButtonElement).disabled).toBe(false)
    await create.trigger('click')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  })
  it('ignores stale detail responses after selecting another ticket', async () => {
    let finish!: (value: unknown) => void
    mocks.detail.mockImplementation((_admin, id) => id === 1 ? new Promise(resolve => { finish = resolve }) : Promise.resolve(detail(2)))
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click')
    await wrapper.findAll('button').find(b => b.text().includes('Problem 2'))!.trigger('click')
    await flushPromises(); finish(detail(1)); await flushPromises()
    expect(wrapper.find('h3').text()).toBe('Problem 2')
    expect(mocks.read).toHaveBeenCalledWith(false, 2, 20)
    expect(mocks.read).not.toHaveBeenCalledWith(false, 1, 10)
  })
  it('does not clear a draft when submitting fails', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.create')!.trigger('click')
    await wrapper.get('input[maxlength="120"]').setValue('Recharge missing')
    await wrapper.get('textarea').setValue('Please check order 123')
    mocks.create.mockRejectedValue({ message: 'Please retry later' })
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Please check order 123')
    expect(wrapper.get('[role="alert"]').text()).toContain('Please retry later')
    const form = mocks.create.mock.calls[0][0] as FormData
    expect(form.get('body')).toBe('Please check order 123')
  })
  it('accepts pasted screenshots, rejects oversized images and releases preview URLs', async () => {
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === 'tickets.create')!.trigger('click')
    const image = new File(['png'], 'screen.png', { type: 'image/png' })
    await wrapper.get('textarea').trigger('paste', { clipboardData: { files: [image] } })
    expect(wrapper.find('img').exists()).toBe(true)
    const tooBig = new File([new Uint8Array(2 * 1024 * 1024 + 1)], 'large.png', { type: 'image/png' })
    await wrapper.get('textarea').trigger('paste', { clipboardData: { files: [tooBig] } })
    expect(wrapper.get('[role="alert"]').text()).toBe('tickets.fileError')
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:preview')
  })
  it('requires reopening a resolved ticket before replying', async () => {
    mocks.detail.mockResolvedValue(detail(1, 'resolved'))
    const wrapper = setup(); await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    expect(wrapper.find('textarea').exists()).toBe(false)
    await wrapper.findAll('button').find(b => b.text() === 'tickets.reopen')!.trigger('click'); await flushPromises()
    expect(mocks.status).toHaveBeenCalledWith(false, 1, 'pending')
  })
  it('uses admin endpoints only in the admin workspace', async () => {
    route.path = '/admin/tickets'; auth.user.role = 'admin'
    const wrapper = setup(); await flushPromises()
    expect(mocks.list).toHaveBeenCalledWith(true, expect.anything())
    expect(wrapper.findAll('button').some(b => b.text() === 'tickets.create')).toBe(false)
    await wrapper.findAll('button').find(b => b.text().includes('Problem 1'))!.trigger('click'); await flushPromises()
    expect(mocks.detail).toHaveBeenCalledWith(true, 1)
  })
  it('drops stale list results on account changes', async () => {
    let finish!: (value: unknown) => void
    mocks.list.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const wrapper = setup()
    auth.user = { id: 2, role: 'user' }
    mocks.list.mockResolvedValue({ items: [], total: 0 })
    await flushPromises()
    finish({ items: [ticket(1)], total: 1 }); await flushPromises()
    expect(wrapper.text()).not.toContain('Problem 1')
  })
})
