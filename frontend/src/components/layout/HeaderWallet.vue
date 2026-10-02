<template>
  <div
    ref="walletRef"
    class="header-wallet relative"
    @pointerenter="handlePointerEnter"
    @pointerleave="handlePointerLeave"
    @focusout="handleFocusOut"
  >
    <div class="wallet-badge" :class="{ 'wallet-badge-open': open }">
      <button
        ref="triggerRef"
        type="button"
        class="wallet-trigger"
        :aria-label="`${t('common.walletDetails')} · ${money(availableBalance)}`"
        :aria-expanded="open"
        aria-controls="header-wallet-details"
        @click="toggle"
      >
        <span class="wallet-icon"><Icon name="creditCard" size="md" aria-hidden="true" /></span>
        <span class="flex min-w-0 flex-col items-start gap-0.5">
          <span class="text-[10px] font-medium leading-none text-gray-500 dark:text-dark-400">{{ t('common.balance') }}</span>
          <span class="wallet-badge-amount">{{ money(availableBalance) }}</span>
        </span>
        <Icon name="chevronDown" size="xs" class="wallet-chevron" :class="{ 'rotate-180': open }" aria-hidden="true" />
      </button>
      <RouterLink v-if="rechargeEnabled" to="/purchase" class="wallet-quick-recharge" @click="close">
        <Icon name="plus" size="xs" aria-hidden="true" />
        {{ t('nav.recharge') }}
      </RouterLink>
    </div>

    <Transition name="wallet">
      <div v-if="open" class="wallet-popover">
        <section id="header-wallet-details" class="wallet-panel" :aria-label="t('common.walletDetails')">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('common.availableBalance') }}</span>
                <span class="wallet-currency">USD</span>
              </div>
              <p class="wallet-amount" data-testid="wallet-available">{{ money(availableBalance) }}</p>
            </div>
            <span class="wallet-panel-icon"><Icon name="creditCard" size="lg" aria-hidden="true" /></span>
          </div>

          <dl class="wallet-stats">
            <div class="min-w-0">
              <dt>{{ t('common.frozenBalance') }}</dt>
              <dd data-testid="wallet-frozen" :class="{ 'wallet-frozen-active': safeFrozenBalance > 0 }">{{ money(safeFrozenBalance) }}</dd>
            </div>
            <div class="min-w-0">
              <dt>{{ t('common.totalBalance') }}</dt>
              <dd data-testid="wallet-total">{{ money(totalBalance) }}</dd>
            </div>
          </dl>

          <div v-if="rechargeEnabled || redeemEnabled || ordersEnabled" class="wallet-actions">
            <RouterLink v-if="rechargeEnabled" to="/purchase" class="wallet-recharge" @click="close">
              <span class="flex items-center gap-2"><Icon name="plus" size="sm" aria-hidden="true" />{{ t('dashboard.balanceNotice.recharge') }}</span>
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </RouterLink>
            <div v-if="redeemEnabled || ordersEnabled" class="wallet-shortcuts">
              <RouterLink v-if="redeemEnabled" to="/redeem" class="wallet-shortcut" @click="close">
                <Icon name="gift" size="sm" aria-hidden="true" />{{ t('nav.redeem') }}
              </RouterLink>
              <RouterLink v-if="ordersEnabled" to="/orders" class="wallet-shortcut" @click="close">
                <Icon name="document" size="sm" aria-hidden="true" />{{ t('nav.myOrders') }}
              </RouterLink>
            </div>
          </div>
        </section>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  open: boolean
  balance: number
  frozenBalance: number
  rechargeEnabled: boolean
  redeemEnabled: boolean
  ordersEnabled: boolean
}>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()
const { t } = useI18n()
const route = useRoute()
const walletRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const availableBalance = computed(() => Number.isFinite(props.balance) ? props.balance : 0)
const safeFrozenBalance = computed(() => Number.isFinite(props.frozenBalance) ? props.frozenBalance : 0)
const totalBalance = computed(() => availableBalance.value + safeFrozenBalance.value)
const moneyFormatter = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })
const money = (value: number) => moneyFormatter.format(value)
let closeTimer: ReturnType<typeof setTimeout> | undefined
let openedByHover = false

function clearCloseTimer() {
  if (closeTimer !== undefined) clearTimeout(closeTimer)
  closeTimer = undefined
}

function close() {
  clearCloseTimer()
  openedByHover = false
  emit('update:open', false)
}

function toggle() {
  clearCloseTimer()
  // Clicking an already-hovered trigger pins the panel instead of closing it.
  if (props.open && openedByHover) {
    openedByHover = false
    return
  }
  emit('update:open', !props.open)
}

function handlePointerEnter(event: PointerEvent) {
  if (event.pointerType !== 'mouse') return
  clearCloseTimer()
  if (!props.open) {
    openedByHover = true
    emit('update:open', true)
  }
}

function handlePointerLeave(event: PointerEvent) {
  if (event.pointerType !== 'mouse' || walletRef.value?.contains(document.activeElement)) return
  clearCloseTimer()
  closeTimer = setTimeout(close, 180)
}

function handleFocusOut(event: FocusEvent) {
  if (!walletRef.value?.contains(event.relatedTarget as Node | null)) close()
}

function handleOutsideClick(event: MouseEvent) {
  if (props.open && !walletRef.value?.contains(event.target as Node)) close()
}

function handleEscape(event: KeyboardEvent) {
  if (event.key !== 'Escape' || !props.open) return
  const restoreFocus = walletRef.value?.contains(document.activeElement)
  close()
  if (restoreFocus) triggerRef.value?.focus()
}

watch(() => route.fullPath, close)
watch(() => props.open, value => {
  if (!value) {
    clearCloseTimer()
    openedByHover = false
  }
})

onMounted(() => {
  document.addEventListener('click', handleOutsideClick)
  document.addEventListener('keydown', handleEscape)
})
onBeforeUnmount(() => {
  clearCloseTimer()
  document.removeEventListener('click', handleOutsideClick)
  document.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
.wallet-badge {
  @apply flex items-center gap-1 rounded-2xl border border-gray-200/70 bg-white/70 p-1 transition-colors dark:border-dark-600/70 dark:bg-dark-800/60;
  box-shadow: 0 2px 8px rgb(15 23 42 / 3%);
}
.wallet-badge:hover,
.wallet-badge-open {
  @apply border-primary-200/80 dark:border-primary-800/70;
}
.wallet-trigger {
  @apply flex min-h-9 items-center gap-2 rounded-xl px-1.5 text-left;
}
.wallet-icon {
  @apply flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-400;
}
.wallet-badge-amount {
  @apply text-sm font-semibold leading-tight text-gray-800 dark:text-gray-100;
  font-variant-numeric: tabular-nums;
}
.wallet-chevron {
  @apply ml-0.5 text-gray-400 transition-transform dark:text-dark-400;
}
.wallet-quick-recharge {
  @apply flex min-h-8 shrink-0 items-center gap-1 rounded-xl bg-primary-50 px-2.5 text-xs font-semibold text-primary-700 transition-colors hover:bg-primary-100 dark:bg-primary-900/25 dark:text-primary-300 dark:hover:bg-primary-900/40;
}
.wallet-popover {
  @apply absolute right-0 top-full z-50 pt-2.5;
  width: min(320px, calc(100vw - 32px));
  transform-origin: top right;
}
.wallet-panel {
  @apply rounded-[22px] border border-white/90 p-5 backdrop-blur-2xl dark:border-dark-600/80;
  background: linear-gradient(145deg, rgb(249 253 252 / 97%), rgb(255 255 255 / 96%) 58%);
  box-shadow: 0 16px 44px -12px rgb(15 23 42 / 18%), 0 0 0 1px rgb(148 163 184 / 13%);
}
.dark .wallet-panel {
  background: linear-gradient(145deg, rgb(30 44 49 / 97%), rgb(30 41 55 / 97%) 58%);
  box-shadow: 0 16px 44px -12px rgb(0 0 0 / 40%);
}
.wallet-currency {
  @apply rounded-md border border-gray-200/70 bg-white/70 px-1.5 py-0.5 text-[9px] font-semibold tracking-wide text-gray-400 dark:border-dark-600 dark:bg-dark-800/60 dark:text-dark-400;
}
.wallet-amount {
  @apply mt-2 text-[32px] font-semibold leading-tight tracking-tight text-gray-900 dark:text-white;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}
.wallet-panel-icon {
  @apply flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-primary-100/80 bg-primary-50/80 text-primary-600 dark:border-primary-800/40 dark:bg-primary-900/20 dark:text-primary-400;
}
.wallet-stats {
  @apply mt-5 grid grid-cols-2 gap-3 rounded-2xl border border-gray-100 bg-gray-50/70 px-4 py-3 dark:border-dark-600/60 dark:bg-dark-900/30;
}
.wallet-stats > div + div {
  @apply border-l border-gray-200/60 pl-4 dark:border-dark-600/60;
}
.wallet-stats dt {
  @apply text-[11px] text-gray-500 dark:text-dark-400;
}
.wallet-stats dd {
  @apply mt-1.5 text-sm font-semibold text-gray-700 dark:text-gray-200;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}
.wallet-stats .wallet-frozen-active {
  @apply text-amber-700 dark:text-amber-300;
}
.wallet-actions {
  @apply mt-4;
}
.wallet-recharge {
  @apply flex min-h-10 items-center justify-between gap-2 rounded-xl border border-primary-200/70 bg-primary-50 px-3.5 text-sm font-semibold text-primary-700 transition-colors hover:border-primary-300 hover:bg-primary-100 dark:border-primary-800/70 dark:bg-primary-900/30 dark:text-primary-200 dark:hover:bg-primary-900/50;
}
.wallet-shortcuts {
  @apply mt-2 flex gap-2;
}
.wallet-shortcut {
  @apply flex min-h-10 min-w-0 flex-1 items-center justify-center gap-2 rounded-xl text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100/80 hover:text-gray-800 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-gray-100;
}
.header-wallet :is(button, a):focus-visible {
  @apply outline-none ring-2 ring-primary-400 ring-offset-2 dark:ring-offset-dark-800;
}
.wallet-enter-active,
.wallet-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}
.wallet-enter-from,
.wallet-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
@media (prefers-reduced-motion: reduce) {
  .header-wallet *, .wallet-enter-active, .wallet-leave-active {
    transition: none !important;
  }
}
</style>
