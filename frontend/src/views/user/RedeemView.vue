<template>
  <AppLayout>
    <div data-testid="redeem-workspace" class="mx-auto w-full max-w-[1440px] space-y-6">
      <div class="grid items-stretch gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
        <!-- Redeem form is the primary action; account and shop stay alongside it. -->
        <section class="redeem-surface flex min-w-0 flex-col p-5 sm:p-8 xl:col-start-2 xl:row-start-1" aria-labelledby="redeem-form-title">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <h2 id="redeem-form-title" class="text-xl font-semibold tracking-tight text-gray-900 dark:text-white sm:text-2xl">
                {{ t('redeem.workspace.formTitle') }}
              </h2>
              <p class="mt-2 text-sm leading-relaxed text-gray-500 dark:text-dark-400">
                {{ t('redeem.workspace.formHint') }}
              </p>
            </div>
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-primary-100 bg-primary-50 text-primary-600 dark:border-primary-800/60 dark:bg-primary-900/30 dark:text-primary-300">
              <Icon name="gift" size="lg" aria-hidden="true" />
            </div>
          </div>

          <form @submit.prevent="handleRedeem" :aria-busy="submitting" class="my-7 space-y-4 sm:my-8">
            <div>
              <div class="mb-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
                <label for="code" class="text-sm font-medium text-gray-700 dark:text-gray-200">
                  {{ t('redeem.redeemCodeLabel') }}
                </label>
                <span id="redeem-code-hint" class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('redeem.redeemCodeHint') }}
                </span>
              </div>
              <div class="relative">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="key" size="md" class="text-primary-500 dark:text-primary-400" aria-hidden="true" />
                </div>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  autocomplete="off"
                  autocapitalize="off"
                  :spellcheck="false"
                  aria-describedby="redeem-code-hint"
                  :aria-invalid="!!errorMessage"
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="input min-h-16 rounded-xl bg-white/80 pl-12 pr-4 font-mono text-base dark:bg-dark-950/40 sm:text-lg"
                />
              </div>
            </div>

            <button
              type="submit"
              data-testid="redeem-submit"
              :disabled="!redeemCode.trim() || submitting"
              class="btn btn-primary min-h-12 w-full gap-2 rounded-xl"
            >
              <span v-if="submitting" class="h-5 w-5 animate-spin rounded-full border-2 border-white/35 border-t-white" aria-hidden="true"></span>
              <Icon v-else name="checkCircle" size="md" aria-hidden="true" />
              {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
              <Icon v-if="!submitting" name="arrowRight" size="sm" aria-hidden="true" />
            </button>
          </form>

          <!-- Success Message -->
          <transition name="fade">
            <div
              v-if="redeemResult"
              role="status"
              class="card mb-6 border-emerald-200 bg-emerald-50 dark:border-emerald-800/50 dark:bg-emerald-900/20"
            >
              <div class="p-6">
                <div class="flex items-start gap-4">
                  <div
                    class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30"
                  >
                    <Icon name="checkCircle" size="md" class="text-emerald-600 dark:text-emerald-400" />
                  </div>
                  <div class="flex-1">
                    <h3 class="text-sm font-semibold text-emerald-800 dark:text-emerald-300">
                      {{ t('redeem.redeemSuccess') }}
                    </h3>
                    <div class="mt-2 text-sm text-emerald-700 dark:text-emerald-400">
                      <p>{{ redeemResult.message }}</p>
                      <div class="mt-3 space-y-1">
                        <p v-if="redeemResult.type === 'balance'" class="font-medium">
                          {{ t('redeem.added') }}: ${{ redeemResult.value.toFixed(2) }}
                        </p>
                        <p v-else-if="redeemResult.type === 'concurrency'" class="font-medium">
                          {{ t('redeem.added') }}: {{ redeemResult.value }}
                          {{ t('redeem.concurrentRequests') }}
                        </p>
                        <p v-else-if="redeemResult.type === 'subscription'" class="font-medium">
                          {{ t('redeem.subscriptionAssigned') }}
                          <span v-if="redeemResult.group_name"> - {{ redeemResult.group_name }}</span>
                          <span v-if="redeemResult.validity_days">
                            ({{
                              t('redeem.subscriptionDays', { days: redeemResult.validity_days })
                            }})</span
                          >
                        </p>
                        <p v-if="redeemResult.new_balance !== undefined">
                          {{ t('redeem.newBalance') }}:
                          <span class="font-semibold">${{ redeemResult.new_balance.toFixed(2) }}</span>
                        </p>
                        <p v-if="redeemResult.new_concurrency !== undefined">
                          {{ t('redeem.newConcurrency') }}:
                          <span class="font-semibold"
                            >{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</span
                          >
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </transition>

          <!-- Error Message -->
          <transition name="fade">
            <div
              v-if="errorMessage"
              role="alert"
              class="card mb-6 border-red-200 bg-red-50 dark:border-red-800/50 dark:bg-red-900/20"
            >
              <div class="p-6">
                <div class="flex items-start gap-4">
                  <div
                    class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-red-100 dark:bg-red-900/30"
                  >
                    <Icon
                      name="exclamationCircle"
                      size="md"
                      class="text-red-600 dark:text-red-400"
                    />
                  </div>
                  <div class="flex-1">
                    <h3 class="text-sm font-semibold text-red-800 dark:text-red-300">
                      {{ t('redeem.redeemFailed') }}
                    </h3>
                    <p class="mt-2 text-sm text-red-700 dark:text-red-400">
                      {{ errorMessage }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </transition>

          <div class="flex flex-wrap gap-2">
            <span v-for="benefit in codeBenefits" :key="benefit.label" class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200/65 bg-gray-50/70 px-3 py-2 text-xs font-medium text-gray-600 dark:border-white/10 dark:bg-white/5 dark:text-dark-300">
              <Icon :name="benefit.icon" size="sm" aria-hidden="true" />
              {{ t(benefit.label) }}
            </span>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">{{ t('redeem.workspace.benefitsHint') }}</p>

          <div class="mt-auto pt-7">
            <div class="flex items-start gap-3 border-t border-gray-200/65 pt-5 dark:border-white/10">
              <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" aria-hidden="true" />
              <div class="min-w-0 space-y-1.5 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
                <h3 class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('redeem.workspace.helpTitle') }}</h3>
                <p>{{ t('redeem.codeRule1') }}</p>
                <p>{{ t('redeem.codeRule3') }}</p>
                <p v-if="contactInfo" data-testid="redeem-contact" class="break-words text-primary-700 dark:text-primary-300">{{ contactInfo }}</p>
              </div>
            </div>
          </div>
        </section>

        <aside class="flex min-w-0 flex-col gap-6 xl:col-start-1 xl:row-start-1">
          <section class="redeem-account relative isolate flex flex-col overflow-hidden rounded-2xl border border-white/80 p-6 text-gray-900 shadow-glass-sm backdrop-blur-xl dark:border-white/10 dark:text-gray-100 xl:flex-1" aria-labelledby="redeem-account-title">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-primary-100/80 bg-primary-50/80 text-primary-600 dark:border-white/10 dark:bg-white/5 dark:text-primary-200">
                <Icon name="creditCard" size="md" aria-hidden="true" />
              </div>
              <div class="min-w-0">
                <h2 id="redeem-account-title" class="text-xs text-gray-500 dark:text-dark-300">{{ t('redeem.workspace.account') }}</h2>
                <p v-if="user?.username" class="mt-0.5 truncate text-base font-semibold">{{ user.username }}</p>
              </div>
            </div>
            <dl class="mt-5 grid flex-1 auto-rows-fr gap-3">
              <div class="rounded-xl border border-gray-200/60 bg-white/70 p-4 dark:border-white/10 dark:bg-white/5">
                <dt class="text-sm font-medium text-gray-600 dark:text-dark-300">{{ t('redeem.currentBalance') }}</dt>
                <dd data-testid="redeem-account-balance" class="mt-3 break-words text-[2.75rem] font-bold leading-none tracking-tight text-gray-800 tabular-nums dark:text-gray-100">
                  ${{ user?.balance?.toFixed(2) || '0.00' }}
                </dd>
              </div>
              <div class="rounded-xl border border-gray-200/60 bg-white/70 p-4 dark:border-white/10 dark:bg-white/5">
                <dt class="text-sm font-medium text-gray-600 dark:text-dark-300">{{ t('redeem.concurrency') }}</dt>
                <dd class="mt-3 break-words text-[2.75rem] font-bold leading-none tracking-tight text-gray-800 tabular-nums dark:text-gray-100">
                  <span data-testid="redeem-account-concurrency">{{ user?.concurrency || 0 }}</span>
                  <span class="ml-2 text-xs font-normal tracking-normal text-gray-500 dark:text-dark-300">{{ t('redeem.requests') }}</span>
                </dd>
              </div>
            </dl>
          </section>

          <section v-if="purchaseUrl" data-testid="redeem-purchase-card" class="redeem-surface p-6" aria-labelledby="redeem-purchase-title">
            <div class="flex items-center gap-2.5">
              <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="externalLink" size="md" aria-hidden="true" />
              </span>
              <h2 id="redeem-purchase-title" class="text-base font-semibold text-gray-900 dark:text-white">{{ t('redeem.workspace.purchaseTitle') }}</h2>
            </div>
            <p class="mt-3 text-sm leading-relaxed text-gray-500 dark:text-dark-400">{{ t('redeem.workspace.purchaseHint') }}</p>
            <a
              data-testid="redeem-purchase-link"
              :href="purchaseUrl"
              target="_blank"
              rel="noopener noreferrer"
              :title="t('redeem.workspace.opensNewTab')"
              class="mt-5 flex min-h-11 w-full items-center justify-between gap-3 rounded-xl border border-primary-200 bg-primary-50/70 px-4 py-3 text-sm font-semibold text-primary-700 transition-colors hover:border-primary-400 hover:bg-primary-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:border-primary-800/70 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:bg-primary-900/40 dark:focus-visible:ring-offset-dark-900"
            >
              {{ t('redeem.workspace.purchaseButton') }}
              <Icon name="externalLink" size="sm" aria-hidden="true" />
            </a>
          </section>
        </aside>
      </div>

      <!-- Recent Activity -->
      <section class="redeem-surface" aria-labelledby="redeem-history-title">
        <div class="flex items-center gap-3 border-b border-gray-200/60 px-5 py-5 dark:border-white/10 sm:px-7">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100/80 text-gray-500 dark:bg-white/5 dark:text-dark-400"><Icon name="clock" size="md" aria-hidden="true" /></span>
          <div>
            <h2 id="redeem-history-title" class="text-base font-semibold text-gray-900 dark:text-white">{{ t('redeem.recentActivity') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('redeem.workspace.historyHint') }}</p>
          </div>
        </div>
        <div class="p-5 sm:px-7">
          <!-- Loading State -->
          <div v-if="loadingHistory" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>

          <!-- History List -->
          <div v-else-if="history.length > 0" class="space-y-3">
            <div
              v-for="item in history"
              :key="item.id"
              class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-gray-100/70 bg-white/55 p-4 dark:border-white/5 dark:bg-white/[0.025]"
            >
              <div class="flex min-w-0 items-center gap-3">
                <div
                  :class="[
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'bg-emerald-100 dark:bg-emerald-900/30'
                        : 'bg-red-100 dark:bg-red-900/30'
                      : isSubscriptionType(item.type)
                        ? 'bg-purple-100 dark:bg-purple-900/30'
                        : item.value >= 0
                          ? 'bg-blue-100 dark:bg-blue-900/30'
                          : 'bg-orange-100 dark:bg-orange-900/30'
                  ]"
                >
                  <!-- 余额类型图标 -->
                  <Icon
                    v-if="isBalanceType(item.type)"
                    name="dollar"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : 'text-red-600 dark:text-red-400'
                    "
                  />
                  <!-- 订阅类型图标 -->
                  <Icon
                    v-else-if="isSubscriptionType(item.type)"
                    name="badge"
                    size="md"
                    class="text-purple-600 dark:text-purple-400"
                  />
                  <!-- 并发类型图标 -->
                  <Icon
                    v-else
                    name="bolt"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-orange-600 dark:text-orange-400'
                    "
                  />
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getHistoryItemTitle(item) }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(item.used_at) }}
                  </p>
                </div>
              </div>
              <div class="ml-auto min-w-0 text-right">
                <p
                  :class="[
                    'break-words text-sm font-semibold tabular-nums',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : 'text-red-600 dark:text-red-400'
                      : isSubscriptionType(item.type)
                        ? 'text-purple-600 dark:text-purple-400'
                        : item.value >= 0
                          ? 'text-blue-600 dark:text-blue-400'
                          : 'text-orange-600 dark:text-orange-400'
                  ]"
                >
                  {{ formatHistoryValue(item) }}
                </p>
                <p
                  v-if="!isAdminAdjustment(item.type)"
                  class="font-mono text-xs text-gray-400 dark:text-dark-500"
                >
                  {{ item.code.slice(0, 8) }}...
                </p>
                <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('redeem.adminAdjustment') }}
                </p>
                <!-- Display notes for admin adjustments -->
                <p
                  v-if="item.notes"
                  class="mt-1 text-xs text-gray-500 dark:text-dark-400 italic max-w-[200px] truncate"
                  :title="item.notes"
                >
                  {{ item.notes }}
                </p>
              </div>
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('redeem.historyWillAppear') }}
            </p>
          </div>
          <div class="mt-5 flex flex-col gap-4 border-t border-gray-200/60 pt-5 text-sm text-gray-500 dark:border-white/10 dark:text-dark-400 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex flex-wrap items-center justify-between gap-3 sm:justify-start sm:gap-6">
              <span>{{ t('common.total') }}: {{ historyTotal }} {{ t('pagination.results') }}</span>
              <label class="flex items-center gap-2">
                {{ t('pagination.perPage') }}
                <select
                  v-model="historyPageSize"
                  class="input w-20"
                  :disabled="loadingHistory || submitting"
                  @change="fetchHistory(1)"
                >
                  <option v-for="size in [20, 50, 100]" :key="size" :value="size">{{ size }}</option>
                </select>
              </label>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:flex sm:gap-2">
              <button
                class="btn btn-secondary"
                :disabled="loadingHistory || submitting || historyPage <= 1"
                @click="fetchHistory(historyPage - 1)"
              >{{ t('pagination.previous') }}</button>
              <button
                class="btn btn-secondary"
                :disabled="loadingHistory || submitting || historyPage * historyPageSize >= historyTotal"
                @click="fetchHistory(historyPage + 1)"
              >{{ t('pagination.next') }}</button>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import { sanitizeUrl } from '@/utils/url'
import { DEFAULT_REDEEM_PURCHASE_URL } from '@/config/redeem'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)
const codeBenefits = [
  { icon: 'creditCard', label: 'redeem.workspace.balanceType' },
  { icon: 'bolt', label: 'redeem.workspace.concurrencyType' },
  { icon: 'badge', label: 'redeem.workspace.subscriptionType' },
] as const

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
} | null>(null)
const errorMessage = ref('')

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
let historyRequest = 0
let loadedHistoryPageSize = 20
const contactInfo = ref('')
const configuredPurchaseUrl = ref('')
const purchaseUrl = computed(() => sanitizeUrl(configuredPurchaseUrl.value))

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async (page = 1) => {
  const request = ++historyRequest
  const pageSize = historyPageSize.value
  loadingHistory.value = true
  try {
    const result = await redeemAPI.getHistory(page, pageSize)
    if (request !== historyRequest) return
    history.value = result.items
    historyTotal.value = result.total
    historyPage.value = page
    historyPageSize.value = pageSize
    loadedHistoryPageSize = pageSize
  } catch (error) {
    if (request !== historyRequest) return
    historyPageSize.value = loadedHistoryPageSize
    appStore.showError(t('redeem.historyLoadFailed'))
    console.error('Failed to fetch history:', error)
  } finally {
    if (request === historyRequest) loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (submitting.value) return
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())

    redeemResult.value = result

    // Refresh user data to get updated balance/concurrency
    try {
      await authStore.refreshUser()
    } catch (error) {
      console.error('Failed to refresh user after redeem:', error)
      appStore.showWarning(t('redeem.userRefreshFailed'))
    }

    // If subscription type, immediately refresh subscription status
    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true) // force refresh
      } catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    // Clear the input
    redeemCode.value = ''

    // Refresh history
    await fetchHistory()

    // Show success toast
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('redeem.failedToRedeem')

    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
    configuredPurchaseUrl.value = settings.redeem_purchase_url ?? DEFAULT_REDEEM_PURCHASE_URL
  } catch (error) {
    console.error('Failed to load redemption page settings:', error)
  }
})
</script>

<style scoped>
.redeem-surface {
  @apply rounded-2xl border border-white/80 bg-white/80 shadow-glass-sm backdrop-blur-xl dark:border-white/10 dark:bg-dark-900/70;
}

.redeem-account {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(244, 250, 249, 0.92));
}

.dark .redeem-account {
  background: linear-gradient(135deg, rgba(28, 40, 52, 0.96), rgba(31, 49, 55, 0.92));
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
