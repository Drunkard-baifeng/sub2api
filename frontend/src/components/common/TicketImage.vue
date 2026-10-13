<template>
  <button type="button" class="block overflow-hidden rounded-xl border border-gray-200 bg-white/60 dark:border-dark-700 dark:bg-dark-900" :aria-label="t('tickets.preview')" @click="src ? (expanded = !expanded) : load()">
    <img v-if="src" :src="src" :alt="t('tickets.screenshot')" :class="expanded ? 'max-h-[70vh] max-w-full object-contain' : 'h-28 max-w-[220px] object-contain'" />
    <span v-else class="block p-4 text-xs text-gray-500">{{ t(failed ? 'tickets.imageError' : 'tickets.loading') }}</span>
  </button>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ticketsAPI } from '@/api/tickets'
const props = defineProps<{ admin: boolean; ticketId: number; attachmentId: number }>()
const { t } = useI18n()
const src = ref(''), failed = ref(false), expanded = ref(false)
let generation = 0
async function load() {
  const token = ++generation
  if (src.value) URL.revokeObjectURL(src.value)
  src.value = ''; failed.value = false; expanded.value = false
  try {
    const blob = await ticketsAPI.attachment(props.admin, props.ticketId, props.attachmentId)
    if (token === generation) src.value = URL.createObjectURL(blob)
  } catch { if (token === generation) failed.value = true }
}
watch(() => [props.admin, props.ticketId, props.attachmentId], load, { immediate: true })
onUnmounted(() => { generation++; if (src.value) URL.revokeObjectURL(src.value) })
</script>
