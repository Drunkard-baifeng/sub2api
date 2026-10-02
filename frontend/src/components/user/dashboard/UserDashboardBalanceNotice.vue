<template>
  <div
    v-if="showNotice"
    role="status"
    data-testid="dashboard-balance-notice"
    class="flex flex-col gap-4 rounded-2xl border p-4 shadow-sm backdrop-blur-xl sm:flex-row sm:items-center sm:justify-between sm:px-5"
    :class="isBalanceEmpty
      ? 'border-amber-200/70 bg-amber-50/80 dark:border-amber-700/40 dark:bg-amber-950/20'
      : 'border-primary-200/70 bg-primary-50/80 dark:border-primary-700/40 dark:bg-primary-950/20'"
  >
    <div class="flex min-w-0 items-start gap-3 sm:items-center">
      <div
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border bg-white/80 dark:bg-dark-800/60"
        :class="isBalanceEmpty
          ? 'border-amber-200/80 text-amber-600 dark:border-amber-700/40 dark:text-amber-400'
          : 'border-primary-200/80 text-primary-600 dark:border-primary-700/40 dark:text-primary-400'"
      >
        <Icon :name="isBalanceEmpty ? 'exclamationTriangle' : 'creditCard'" size="lg" aria-hidden="true" />
      </div>
      <div class="min-w-0">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t(isBalanceEmpty ? 'dashboard.balanceNotice.title' : 'dashboard.balanceNotice.balanceTitle') }}
        </h2>
        <p
          class="mt-1 text-sm leading-relaxed"
          :class="isBalanceEmpty ? 'text-amber-800 dark:text-amber-200/90' : 'text-primary-800 dark:text-primary-200/90'"
        >
          {{ t(isBalanceEmpty ? 'dashboard.balanceNotice.description' : 'dashboard.balanceNotice.balanceDescription', { balance: formattedBalance }) }}
        </p>
      </div>
    </div>
    <RouterLink
      to="/purchase"
      class="btn btn-primary min-h-11 w-full shrink-0 gap-2 whitespace-nowrap sm:w-auto"
    >
      {{ t('dashboard.balanceNotice.recharge') }}
      <Icon name="arrowRight" size="sm" aria-hidden="true" />
    </RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

const props = defineProps<{
  balance: number
  isSimple: boolean
}>()

const { t } = useI18n()
const appStore = useAppStore()
const isBalanceEmpty = computed(() => props.balance <= 0)

// 充值可用时常驻显示；余额只决定提示样式，不决定入口是否出现。
// 等设置加载完成再展示，避免出现已关闭的充值入口；简易模式不使用余额计费。
const showNotice = computed(() => {
  const settings = appStore.cachedPublicSettings
  return !props.isSimple
    && Number.isFinite(props.balance)
    && appStore.publicSettingsLoaded
    && resolveFeatureFlag(settings, FeatureFlags.payment)
    && settings?.payment_balance_disabled !== true
})

const formattedBalance = computed(() => new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
}).format(props.balance))
</script>
