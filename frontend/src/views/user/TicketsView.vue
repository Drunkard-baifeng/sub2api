<template>
  <AppLayout>
    <div class="ticket-page">
      <div v-if="error" role="alert" class="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>
      <p class="mb-3 px-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('tickets.retentionNotice', { days: retentionDays }) }}</p>
      <div class="ticket-workspace">
        <section class="ticket-list" :class="(detail || creating || loadingDetail) ? 'hidden lg:flex' : 'flex'">
          <div class="ticket-list-toolbar">
            <div class="flex items-center justify-between gap-3">
              <div class="flex min-w-0 items-center gap-2.5">
                <h2 class="font-semibold text-gray-900 dark:text-white">{{ t(admin ? 'tickets.allTickets' : 'tickets.myTickets') }}</h2>
                <span class="ticket-count">{{ total }}</span>
              </div>
              <button v-if="!admin" type="button" class="btn btn-primary gap-1.5 rounded-xl px-3 py-2 text-xs" :disabled="busy || creating" @click="startNew">
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
                {{ t('tickets.create') }}
              </button>
            </div>
            <div class="mt-5 flex gap-2">
              <select v-model="filter" class="input min-w-0 flex-1 text-sm" :aria-label="t('tickets.all')" @change="resetList">
                <option value="">{{ t('tickets.all') }}</option>
                <option v-for="s in statuses" :key="s" :value="s">{{ t('tickets.statuses.' + s) }}</option>
              </select>
              <button type="button" class="ticket-refresh" :title="t('tickets.refresh')" :aria-label="t('tickets.refresh')" :disabled="loadingList || busy" @click="refresh">
                <svg class="h-4 w-4" :class="{ 'animate-spin': loadingList }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M20 7v5h-5M4 17v-5h5M6.1 7a7 7 0 0 1 11.6-1L20 9M4 15l2.3 3a7 7 0 0 0 11.6-1" /></svg>
              </button>
            </div>
            <label class="mt-3 flex w-fit cursor-pointer items-center gap-2 text-xs text-gray-500 dark:text-dark-300"><input v-model="unreadOnly" type="checkbox" class="rounded border-gray-300 text-primary-500 focus:ring-primary-500" @change="resetList" />{{ t('tickets.unreadOnly') }}</label>
          </div>
          <div class="ticket-list-scroll">
            <p v-if="loadingList && !items.length" class="p-8 text-center text-sm text-gray-500">{{ t('tickets.loading') }}</p>
            <div v-else-if="!items.length" class="px-6 py-12 text-center">
              <svg class="mx-auto mb-3 h-8 w-8 text-gray-300 dark:text-dark-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><path d="M4 4h16v16H4zM4 14h4l2 3h4l2-3h4M8 8h8" /></svg>
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('tickets.empty') }}</p>
            </div>
            <button v-for="ticket in items" :key="ticket.id" type="button" class="ticket-list-item" :class="{ 'ticket-list-item-active': selected === ticket.id }" :aria-pressed="selected === ticket.id" :disabled="busy" @click="selectTicket(ticket.id)">
              <div class="mb-2 flex items-center justify-between gap-2 text-[11px] text-gray-400 dark:text-dark-400">
                <span>#{{ ticket.id }} · {{ t('tickets.categories.' + ticket.category) }}</span>
                <span v-if="ticket.unread" class="h-2 w-2 shrink-0 rounded-full bg-primary-500" :aria-label="t('tickets.unread')"></span>
              </div>
              <p class="break-words text-sm font-semibold leading-6 text-gray-800 dark:text-dark-100">{{ ticket.subject }}</p>
              <p v-if="admin" class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ ticket.email }}</p>
              <div class="mt-3 flex flex-wrap items-center justify-between gap-2 text-[11px] text-gray-400 dark:text-dark-400">
                <span :class="statusClass(ticket.status)" class="rounded-md px-2 py-1">{{ t('tickets.statuses.' + ticket.status) }}</span>
                <span>{{ date(ticket.updated_at) }}</span>
              </div>
            </button>
          </div>
          <div class="ticket-pagination">
            <button type="button" class="btn btn-ghost px-2 text-xs" :disabled="page <= 1 || loadingList" @click="page--; loadList()">{{ t('tickets.prev') }}</button>
            <span>{{ page }} · {{ t('tickets.count', { count: total }) }}</span>
            <button type="button" class="btn btn-ghost px-2 text-xs" :disabled="page * 20 >= total || loadingList" @click="page++; loadList()">{{ t('tickets.next') }}</button>
          </div>
        </section>

        <section class="ticket-detail" :class="(!detail && !creating && !loadingDetail) ? 'hidden lg:flex' : 'flex'">
          <div v-if="creating || detail || loadingDetail" class="ticket-mobile-back">
            <button type="button" class="flex items-center gap-1 text-sm text-primary-600 dark:text-primary-400" :disabled="busy" @click="back">
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="m14 6-6 6 6 6" /></svg>{{ t('tickets.back') }}
            </button>
          </div>
          <div v-if="loadingDetail && !detail" class="m-auto p-10 text-sm text-gray-500">{{ t('tickets.loading') }}</div>
          <div v-else-if="!creating && !detail" class="ticket-empty">
            <div class="ticket-empty-icon">
              <svg class="h-9 w-9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M8 9h8M8 13h5M6 3h12a3 3 0 0 1 3 3v11a3 3 0 0 1-3 3H9l-6 2V6a3 3 0 0 1 3-3Z" /></svg>
            </div>
            <h3 class="text-base font-medium text-gray-700 dark:text-dark-200">{{ t('tickets.choose') }}</h3>
            <p class="mt-2 max-w-xs text-sm leading-6 text-gray-400 dark:text-dark-400">{{ t(admin ? 'tickets.adminEmptyHint' : 'tickets.emptyHint') }}</p>
          </div>

          <template v-if="creating">
            <div class="ticket-detail-heading">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('tickets.create') }}</h3>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">{{ t('tickets.newHint') }}</p>
                </div>
                <button type="button" class="btn btn-ghost hidden px-3 text-sm lg:inline-flex" :disabled="busy" @click="back">{{ t('tickets.cancel') }}</button>
              </div>
            </div>
            <div class="grid gap-4 px-5 pt-5 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] sm:px-6">
              <label class="block text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('tickets.subject') }}<input v-model="subject" maxlength="120" class="input mt-2 w-full text-sm" :placeholder="t('tickets.subjectHint')" :disabled="busy" /></label>
              <label class="block text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('tickets.category') }}<select v-model="category" class="input mt-2 w-full text-sm" :disabled="busy"><option v-for="c in categories" :key="c" :value="c">{{ t('tickets.categories.' + c) }}</option></select></label>
            </div>
          </template>

          <template v-if="detail">
            <div class="ticket-detail-heading">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="min-w-0">
                  <p class="mb-1.5 text-xs text-gray-400 dark:text-dark-400">#{{ detail.ticket.id }} · {{ t('tickets.categories.' + detail.ticket.category) }}</p>
                  <h3 class="break-words text-lg font-semibold text-gray-900 dark:text-white">{{ detail.ticket.subject }}</h3>
                  <p v-if="admin" class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ detail.ticket.email }}</p>
                </div>
                <span class="rounded-lg px-3 py-1.5 text-xs" :class="statusClass(detail.ticket.status)">{{ t('tickets.statuses.' + detail.ticket.status) }}</span>
              </div>
              <div class="mt-4 flex flex-wrap items-center gap-2">
                <span class="mr-1 text-xs text-gray-400 dark:text-dark-400">{{ t('tickets.statusChange') }}</span>
                <template v-if="admin && detail.ticket.status !== 'resolved'">
                  <button v-for="s in statuses.filter(s => s !== detail?.ticket.status)" :key="s" type="button" class="btn btn-secondary rounded-lg px-3 py-1.5 text-xs" :disabled="busy" @click="changeStatus(s)">{{ t('tickets.statuses.' + s) }}</button>
                </template>
                <button v-else type="button" class="btn btn-secondary rounded-lg px-3 py-1.5 text-xs" :disabled="busy" @click="changeStatus(detail.ticket.status === 'resolved' ? 'pending' : 'resolved')">{{ t(detail.ticket.status === 'resolved' ? 'tickets.reopen' : 'tickets.resolve') }}</button>
                <button v-if="admin" type="button" class="ml-auto rounded-lg px-3 py-1.5 text-xs text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:opacity-40 dark:hover:bg-red-950/30 dark:hover:text-red-300" :disabled="busy" @click="deleteTarget = detail.ticket.id">{{ t('tickets.delete') }}</button>
              </div>
              <p v-if="detail.ticket.expires_at" class="mt-3 text-[11px] text-gray-400 dark:text-dark-400">{{ t('tickets.expiresAt', { date: date(detail.ticket.expires_at) }) }}</p>
            </div>
            <div ref="conversation" class="ticket-conversation" aria-live="polite">
              <button v-if="detail.has_more" class="mx-auto block text-xs text-primary-600 dark:text-primary-400" :disabled="loadingOlder" @click="loadOlder">{{ t(loadingOlder ? 'tickets.loading' : 'tickets.older') }}</button>
              <div v-for="message in detail.messages" :key="message.id">
                <p v-if="message.status_event" class="text-center text-[11px] leading-5 text-gray-400 dark:text-dark-400">{{ t(message.is_admin ? 'tickets.staff' : 'tickets.you') }} · {{ t('tickets.statusEvent') }} {{ t('tickets.statuses.' + message.status_event) }} · {{ date(message.created_at) }}</p>
                <div v-else class="flex flex-col" :class="message.is_admin === admin ? 'items-end' : 'items-start'">
                  <span class="mb-1.5 text-[11px] text-gray-400 dark:text-dark-400">{{ t(message.is_admin ? 'tickets.staff' : 'tickets.you') }} · {{ date(message.created_at) }}</span>
                  <div class="max-w-[95%] space-y-3 rounded-2xl border px-4 py-3 sm:max-w-[85%]" :class="message.is_admin === admin ? 'rounded-tr-md border-primary-100 bg-primary-50/70 dark:border-primary-900/50 dark:bg-primary-950/25' : 'rounded-tl-md border-gray-100 bg-white/80 dark:border-dark-700 dark:bg-dark-800/60'">
                    <p v-if="message.body" class="whitespace-pre-wrap break-words text-sm leading-7 text-gray-700 dark:text-dark-100 [overflow-wrap:anywhere]">{{ message.body }}</p>
                    <div v-if="message.attachments.length" class="flex flex-wrap gap-2"><TicketImage v-for="attachment in message.attachments" :key="attachment.id" :admin="admin" :ticket-id="detail.ticket.id" :attachment-id="attachment.id" /></div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <form v-if="creating || (detail && detail.ticket.status !== 'resolved')" class="ticket-composer" :class="{ 'ticket-composer-new': creating }" @submit.prevent="submit" @paste="paste">
            <div class="mb-2.5 flex items-center justify-between gap-3 text-xs">
              <label for="ticket-message" class="font-medium text-gray-600 dark:text-dark-300">{{ t(creating ? 'tickets.description' : 'tickets.replyLabel') }}</label>
              <span class="tabular-nums text-gray-400 dark:text-dark-400">{{ body.length }} / 5000</span>
            </div>
            <div class="ticket-editor">
              <textarea id="ticket-message" v-model="body" maxlength="5000" :rows="creating ? 6 : 3" class="ticket-textarea" :aria-label="t('tickets.bodyHint')" :placeholder="t(creating ? 'tickets.bodyHint' : 'tickets.replyHint')" :disabled="busy" />
              <div v-if="files.length" class="flex flex-wrap gap-3 px-3 pb-3" aria-live="polite">
                <div v-for="(file, index) in files" :key="file.url" class="relative">
                  <img :src="file.url" class="h-20 w-24 rounded-lg border border-gray-200 object-cover dark:border-dark-700" :alt="t('tickets.screenshot')" />
                  <button type="button" class="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-gray-700 text-xs text-white shadow-sm hover:bg-red-500" :aria-label="t('tickets.remove')" :disabled="busy" @click="removeFile(index)">×</button>
                </div>
              </div>
              <div class="ticket-editor-tools">
                <button type="button" class="ticket-attach" :disabled="busy" @click="fileInput?.click()">
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="4" /><circle cx="8.5" cy="8.5" r="1.5" /><path stroke-linecap="round" stroke-linejoin="round" d="m21 15-5-5L6 21" /></svg>
                  {{ t('tickets.selectImage') }}
                </button>
                <input ref="fileInput" type="file" class="hidden" accept="image/png,image/jpeg" multiple :aria-label="t('tickets.selectImage')" :disabled="busy" @change="pickFiles" />
                <span class="hidden items-center gap-1.5 text-[11px] text-gray-400 dark:text-dark-400 sm:flex"><kbd class="ticket-key">Ctrl / ⌘ + V</kbd>{{ t('tickets.pasteImage') }}</span>
                <span v-if="files.length" class="ml-auto text-xs tabular-nums text-primary-600 dark:text-primary-400">{{ files.length }} / 3</span>
              </div>
            </div>
            <div class="mt-3 flex items-center justify-between gap-4">
              <p class="text-[11px] leading-5 text-gray-400 dark:text-dark-400">{{ t('tickets.attachmentHint') }}</p>
              <button type="submit" class="btn btn-primary shrink-0 gap-2 rounded-xl px-4 py-2.5 text-sm" :disabled="busy || (!body.trim() && !files.length) || (creating && subject.trim().length < 2)">
                {{ t(busy ? 'tickets.working' : creating ? 'tickets.submit' : 'tickets.send') }}
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="m5 12 14-8-4 16-3-7-7-1Zm7 1 7-9" /></svg>
              </button>
            </div>
            <p v-if="creating" class="mt-4 text-[11px] leading-5 text-gray-400 dark:text-dark-400">{{ t('tickets.quota') }}</p>
          </form>
          <p v-if="detail?.ticket.status === 'resolved'" class="ticket-resolved">{{ t('tickets.resolvedHint') }}</p>
        </section>
      </div>
    </div>
    <ConfirmDialog v-if="admin" :show="deleteTarget !== null" :title="t('tickets.deleteTitle')" :message="t('tickets.deleteMessage', { id: deleteTarget })" :confirm-text="t('tickets.deleteConfirm')" danger @confirm="deleteTicket" @cancel="deleteTarget = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TicketImage from '@/components/common/TicketImage.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { ticketsAPI, notifyTicketsChanged, type Ticket, type TicketDetail, type TicketStatus } from '@/api/tickets'

const { t, locale } = useI18n()
const route = useRoute()
const auth = useAuthStore()
const admin = computed(() => route.path.startsWith('/admin/'))
const statuses: TicketStatus[] = ['pending', 'processing', 'waiting_user', 'resolved']
const categories = ['recharge', 'redemption', 'api', 'other']
const items = ref<Ticket[]>([]), total = ref(0), page = ref(1), filter = ref(''), unreadOnly = ref(false)
const retentionDays = ref(7), deleteTarget = ref<number | null>(null)
const detail = ref<TicketDetail | null>(null), selected = ref<number | null>(null), creating = ref(false)
const loadingList = ref(false), loadingDetail = ref(false), loadingOlder = ref(false), busy = ref(false), error = ref('')
const subject = ref(''), category = ref('other'), body = ref('')
const files = ref<{ file: File; url: string }[]>([])
const conversation = ref<HTMLElement>()
const fileInput = ref<HTMLInputElement>()
let epoch = 0, listRequest = 0, detailRequest = 0
let timer: ReturnType<typeof setInterval> | undefined
const date = (value: string) => new Date(value).toLocaleString(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
function statusClass(status: string) {
  return ({
    pending: 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300',
    processing: 'bg-sky-50 text-sky-700 dark:bg-sky-950/30 dark:text-sky-300',
    waiting_user: 'bg-primary-50 text-primary-700 dark:bg-primary-950/30 dark:text-primary-300',
    resolved: 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400',
  } as Record<string, string>)[status]
}
function showError(e: unknown) { error.value = (e as { message?: string })?.message || t('tickets.error') }
function clearFiles() { files.value.forEach(f => URL.revokeObjectURL(f.url)); files.value = [] }
function clearDraft() { body.value = ''; subject.value = ''; category.value = 'other'; clearFiles() }
function back() { epoch++; detailRequest++; deleteTarget.value = null; selected.value = null; detail.value = null; creating.value = false; loadingDetail.value = false; clearDraft() }
function startNew() { back(); creating.value = true; error.value = '' }
async function loadList() {
  const token = ++listRequest, scope = epoch
  loadingList.value = true
  try {
    const result = await ticketsAPI.list(admin.value, { page: page.value, status: filter.value, unread: unreadOnly.value })
    if (token === listRequest && scope === epoch) { items.value = result.items; total.value = result.total; retentionDays.value = result.retention_days || 7 }
  } catch (e) { if (token === listRequest && scope === epoch) showError(e) }
  finally { if (token === listRequest) loadingList.value = false }
}
function resetList() { page.value = 1; void loadList() }
async function loadDetail(scroll = false) {
  if (!selected.value || loadingOlder.value) return
  const id = selected.value, token = ++detailRequest, scope = epoch, isAdmin = admin.value
  loadingDetail.value = true
  try {
    const result = await ticketsAPI.detail(isAdmin, id)
    if (token !== detailRequest || scope !== epoch || selected.value !== id) return
    const old = detail.value
    // Retain previously loaded history while refreshing the latest page.
    if (old?.ticket.id === id && old.messages.some(m => result.messages.some(n => n.id === m.id))) {
      result.messages = [...old.messages.filter(m => m.id < result.messages[0].id), ...result.messages]
      result.has_more = old.has_more
    }
    detail.value = result
    await nextTick()
    if (scroll && conversation.value) conversation.value.scrollTop = conversation.value.scrollHeight
    if (!document.hidden && scope === epoch && result.ticket.unread && result.ticket.last_message_id) {
      try {
        await ticketsAPI.read(isAdmin, id, result.ticket.last_message_id)
        if (scope === epoch) { notifyTicketsChanged(); const item = items.value.find(v => v.id === id); if (item) item.unread = false }
      } catch { if (scope === epoch) error.value = t('tickets.readError') }
    }
  } catch (e) { if (token === detailRequest && scope === epoch) handleDetailError(e) }
  finally { if (token === detailRequest) loadingDetail.value = false }
}
async function selectTicket(id: number) { back(); selected.value = id; error.value = ''; await loadDetail(true) }
async function loadOlder() {
  if (!detail.value || loadingDetail.value || loadingOlder.value) return
  const scope = epoch, current = detail.value, height = conversation.value?.scrollHeight || 0
  loadingOlder.value = true
  try {
    const result = await ticketsAPI.detail(admin.value, current.ticket.id, current.messages[0]?.id || 0)
    if (scope !== epoch || !detail.value) return
    detail.value.messages = [...result.messages, ...detail.value.messages]
    detail.value.has_more = result.has_more
    await nextTick()
    if (conversation.value) conversation.value.scrollTop += conversation.value.scrollHeight - height
  } catch (e) { if (scope === epoch) handleDetailError(e) }
  finally { loadingOlder.value = false }
}
function handleDetailError(e: unknown) {
  const status = (e as { status?: number; response?: { status?: number } })
  if (status?.status === 404 || status?.response?.status === 404) {
    back(); error.value = t('tickets.expiredOrDeleted'); notifyTicketsChanged(); void loadList()
  } else showError(e)
}
function addFiles(incoming: File[]) {
  if (busy.value) return
  if (files.value.length + incoming.length > 3 || incoming.some(f => !['image/png', 'image/jpeg'].includes(f.type) || f.size > 2 * 1024 * 1024)) { error.value = t('tickets.fileError'); return }
  files.value.push(...incoming.map(file => ({ file, url: URL.createObjectURL(file) })))
}
function pickFiles(event: Event) { const input = event.target as HTMLInputElement; addFiles(Array.from(input.files || [])); input.value = '' }
function paste(event: ClipboardEvent) {
  if (busy.value || !event.clipboardData) return
  const clipboard = event.clipboardData
  const images = clipboard.files.length
    ? Array.from(clipboard.files)
    : Array.from(clipboard.items || [])
      .filter(item => item.kind === 'file' && item.type.startsWith('image/'))
      .map(item => item.getAsFile())
      .filter((file): file is File => file !== null)
  if (images.length) { event.preventDefault(); addFiles(images) }
}
function removeFile(index: number) { URL.revokeObjectURL(files.value[index].url); files.value.splice(index, 1) }
async function submit() {
  if (busy.value || (!body.value.trim() && !files.value.length)) return
  const scope = epoch, isNew = creating.value, id = selected.value, isAdmin = admin.value
  busy.value = true; error.value = ''
  const form = new FormData()
  form.append('body', body.value.trim())
  files.value.forEach(f => form.append('images', f.file))
  try {
    let newId = id
    if (isNew) { form.append('subject', subject.value.trim()); form.append('category', category.value); newId = await ticketsAPI.create(form) }
    else if (id) await ticketsAPI.reply(isAdmin, id, form)
    if (scope !== epoch) return
    clearDraft(); creating.value = false; selected.value = newId
    notifyTicketsChanged()
    await loadDetail(true); await loadList()
  } catch (e) { if (scope === epoch) showError(e) }
  finally { busy.value = false }
}
async function changeStatus(status: TicketStatus) {
  if (!selected.value || busy.value) return
  const scope = epoch
  busy.value = true; error.value = ''
  try {
    await ticketsAPI.status(admin.value, selected.value, status)
    if (scope !== epoch) return
    await loadDetail(); await loadList(); notifyTicketsChanged()
  } catch (e) { if (scope === epoch) showError(e) }
  finally { busy.value = false }
}
async function deleteTicket() {
  if (!admin.value || deleteTarget.value === null || busy.value) return
  const id = deleteTarget.value, scope = epoch
  deleteTarget.value = null; busy.value = true; error.value = ''
  try {
    await ticketsAPI.remove(id)
    if (scope !== epoch) return
    back(); page.value = 1; notifyTicketsChanged(); await loadList()
  } catch (e) { if (scope === epoch) handleDetailError(e) }
  finally { busy.value = false }
}
async function refresh() {
  if (document.hidden || busy.value || loadingDetail.value || loadingList.value || loadingOlder.value) return
  await loadList()
  await loadDetail()
}
watch([admin, () => auth.user?.id], () => { back(); listRequest++; items.value = []; page.value = 1; total.value = 0; error.value = ''; if (auth.user) void loadList() })
onMounted(() => { void loadList(); timer = setInterval(refresh, 20000); document.addEventListener('visibilitychange', refresh) })
onUnmounted(() => { epoch++; listRequest++; detailRequest++; clearFiles(); clearInterval(timer); document.removeEventListener('visibilitychange', refresh) })
</script>

<style scoped>
.ticket-page { @apply mx-auto max-w-[1680px]; }
.ticket-workspace { @apply grid overflow-hidden rounded-2xl border border-gray-200/80 bg-white/80 shadow-sm backdrop-blur-xl dark:border-dark-700/80 dark:bg-dark-900/80; }
.ticket-list { @apply min-h-[460px] min-w-0 flex-col bg-white/40 dark:bg-dark-900/30; }
.ticket-list-toolbar { @apply shrink-0 border-b border-gray-100 p-5 dark:border-dark-700/70; }
.ticket-count { @apply rounded-md bg-gray-100 px-2 py-0.5 text-xs tabular-nums text-gray-500 dark:bg-dark-800 dark:text-dark-400; }
.ticket-refresh { @apply flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-white/70 text-gray-500 transition-colors hover:text-primary-600 disabled:opacity-40 dark:border-dark-700 dark:bg-dark-800/60 dark:text-dark-300; }
.ticket-list-scroll { @apply min-h-0 flex-1 overflow-y-auto p-2; }
.ticket-list-item { @apply mb-1 block w-full rounded-xl border border-transparent px-3 py-4 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:hover:bg-dark-800/70; }
.ticket-list-item-active { @apply border-primary-100 bg-primary-50/80 hover:bg-primary-50 dark:border-primary-900/60 dark:bg-primary-950/25 dark:hover:bg-primary-950/30; }
.ticket-pagination { @apply flex shrink-0 items-center justify-between gap-2 border-t border-gray-100 p-3 text-xs text-gray-400 dark:border-dark-700/70 dark:text-dark-400; }
.ticket-detail { @apply min-h-[520px] min-w-0 flex-col; }
.ticket-mobile-back { @apply shrink-0 border-b border-gray-100 px-5 py-3 dark:border-dark-700/70 lg:hidden; }
.ticket-detail-heading { @apply shrink-0 border-b border-gray-100 px-5 py-5 dark:border-dark-700/70 sm:px-6; }
.ticket-empty { @apply m-auto flex flex-col items-center px-8 py-16 text-center; }
.ticket-empty-icon { @apply mb-6 flex h-20 w-20 items-center justify-center rounded-3xl border border-primary-100/80 bg-gradient-to-br from-white to-primary-50 text-primary-500 shadow-sm dark:border-primary-900/40 dark:from-dark-800 dark:to-primary-950/40 dark:text-primary-400; }
.ticket-conversation { @apply min-h-[240px] flex-1 space-y-5 overflow-y-auto bg-gray-50/40 px-5 py-6 dark:bg-dark-950/25 sm:px-6; max-height: 55vh; }
.ticket-composer { @apply shrink-0 border-t border-gray-100 px-5 py-4 dark:border-dark-700/70 sm:px-6; }
.ticket-composer-new { @apply flex flex-1 flex-col border-t-0 pt-5; }
.ticket-editor { @apply overflow-hidden rounded-xl border border-gray-200 bg-white/70 transition-shadow focus-within:border-primary-400 focus-within:ring-2 focus-within:ring-primary-500/10 dark:border-dark-600/80 dark:bg-dark-800/40 dark:focus-within:border-primary-500; }
.ticket-composer-new .ticket-editor { @apply flex flex-1 flex-col; }
.ticket-textarea { @apply block w-full resize-y border-0 bg-transparent px-3.5 py-3 text-sm leading-6 text-gray-700 outline-none placeholder:text-gray-400 focus:ring-0 disabled:opacity-50 dark:text-dark-100 dark:placeholder:text-dark-500; min-height: 88px; max-height: 200px; }
.ticket-composer-new .ticket-textarea { @apply flex-1; min-height: 180px; max-height: none; resize: none; }
.ticket-editor-tools { @apply flex shrink-0 flex-wrap items-center gap-3 border-t border-gray-100/80 px-3 py-2 dark:border-dark-700/60; }
.ticket-attach { @apply flex items-center gap-1.5 rounded-lg px-1.5 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 disabled:opacity-40 dark:text-primary-400 dark:hover:bg-primary-950/50; }
.ticket-key { @apply rounded border border-gray-200 px-1.5 py-0.5 font-sans text-[10px] dark:border-dark-600; }
.ticket-resolved { @apply shrink-0 border-t border-gray-100 bg-gray-50/50 p-4 text-center text-xs leading-6 text-gray-500 dark:border-dark-700/70 dark:bg-dark-800/30 dark:text-dark-400; }
@media (min-width: 1024px) {
  .ticket-workspace { grid-template-columns: 320px minmax(0, 1fr); height: clamp(620px, calc(100dvh - 128px), 880px); }
  .ticket-list { min-height: 0; @apply border-r border-gray-200/70 dark:border-dark-700/70; }
  .ticket-detail { min-height: 0; overflow-y: auto; }
  .ticket-conversation { min-height: 140px; max-height: none; }
}
@media (min-width: 1536px) {
  .ticket-workspace { grid-template-columns: 360px minmax(0, 1fr); }
}
@media (prefers-reduced-motion: reduce) {
  .ticket-list-item, .ticket-editor, .ticket-refresh, .ticket-attach { transition: none; }
}
</style>
