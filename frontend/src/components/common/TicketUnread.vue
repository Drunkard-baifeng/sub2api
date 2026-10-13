<template>
  <router-link :to="admin ? '/admin/tickets' : '/tickets'" class="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-gray-500 hover:bg-primary-50 hover:text-primary-600 dark:text-dark-300 dark:hover:bg-dark-800" :title="t(admin ? 'tickets.adminTitle' : 'tickets.title')" :aria-label="t(admin ? 'tickets.adminTitle' : 'tickets.title') + (count ? ' · ' + count : '')">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h8M8 14h5M7 3h10a4 4 0 0 1 4 4v9a4 4 0 0 1-4 4H9l-6 2V7a4 4 0 0 1 4-4Z" /></svg>
    <span v-if="count" class="absolute -right-1 -top-1 min-w-4 rounded-full bg-primary-500 px-1 text-center text-[10px] leading-4 text-white">{{ count > 99 ? '99+' : count }}</span>
  </router-link>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { ticketsAPI } from '@/api/tickets'
const { t } = useI18n()
const auth = useAuthStore()
const route = useRoute()
const admin = computed(() => auth.user?.role === 'admin' && route.path.startsWith('/admin'))
const count = ref(0)
let generation = 0
let interval: ReturnType<typeof setInterval> | undefined
async function refresh() {
  if (document.hidden || !auth.user) return
  const token = ++generation
  const identity = auth.user.id
  try {
    const n = await ticketsAPI.unread(admin.value)
    if (token === generation && identity === auth.user?.id) count.value = n
  } catch { /* Keep the last known count; the ticket page exposes API errors. */ }
}
watch([() => auth.user?.id, admin], () => { generation++; count.value = 0; void refresh() })
onMounted(() => {
  void refresh()
  interval = setInterval(refresh, 30000)
  window.addEventListener('tickets-changed', refresh)
  document.addEventListener('visibilitychange', refresh)
})
onUnmounted(() => {
  generation++
  clearInterval(interval)
  window.removeEventListener('tickets-changed', refresh)
  document.removeEventListener('visibilitychange', refresh)
})
</script>
