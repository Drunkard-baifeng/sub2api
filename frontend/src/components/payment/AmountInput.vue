<template>
  <div class="space-y-5">
    <div role="group" :aria-label="t('payment.quickAmounts')" class="grid grid-cols-2 gap-3 lg:grid-cols-3 2xl:grid-cols-4">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          data-testid="recharge-amount-card"
          :aria-pressed="!customMode && modelValue === amt"
          :class="[
            'recharge-amount-card group',
            !customMode && modelValue === amt ? 'recharge-amount-card-selected' : 'recharge-amount-card-idle',
          ]"
          @click="selectAmount(amt)"
        >
          <span class="flex items-center justify-between gap-2">
            <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('payment.amountLabel') }}</span>
            <span class="flex h-5 w-5 items-center justify-center rounded-full border" :class="!customMode && modelValue === amt ? 'border-primary-500 bg-primary-500 text-white' : 'border-gray-200 dark:border-dark-600'">
              <Icon v-if="!customMode && modelValue === amt" name="check" size="xs" aria-hidden="true" />
            </span>
          </span>
          <span class="mt-5 flex flex-wrap items-baseline gap-x-1.5 text-gray-900 dark:text-white">
            <span class="text-base font-medium text-gray-400 dark:text-dark-400">{{ currencySymbol(currency) }}</span>
            <span class="text-3xl font-bold tracking-tight tabular-nums">{{ amt.toLocaleString() }}</span>
          </span>
          <span class="mt-2 block text-xs leading-relaxed text-gray-500 dark:text-dark-400">
            {{ t('payment.creditedBalance') }}
            <span class="font-semibold text-primary-700 dark:text-primary-300">{{ formatCreditedAmount(amt) }}</span>
          </span>
          <span class="mt-5 flex items-center justify-between gap-2 border-t border-gray-100 pt-3 text-xs font-medium text-primary-700 dark:border-white/10 dark:text-primary-300">
            {{ t(!customMode && modelValue === amt ? 'payment.rechargePanel.selected' : 'payment.rechargePanel.selectAmount') }}
            <Icon name="arrowRight" size="sm" aria-hidden="true" />
          </span>
        </button>
        <button
          type="button"
          data-testid="recharge-custom-card"
          :aria-pressed="customMode"
          :class="['recharge-amount-card flex flex-col items-center justify-center text-center', customMode ? 'recharge-amount-card-selected' : 'recharge-amount-card-idle border-dashed']"
          @click="focusCustomAmount"
        >
          <span class="mb-4 flex h-11 w-11 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"><Icon name="plus" size="lg" aria-hidden="true" /></span>
          <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('payment.customAmount') }}</span>
          <span class="mt-2 text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ t('payment.rechargePanel.customHint') }}</span>
        </button>
    </div>

    <!-- Custom Amount Input -->
    <div class="rounded-2xl border border-white/80 bg-white/75 p-5 shadow-glass-sm backdrop-blur-xl dark:border-white/10 dark:bg-dark-900/65">
      <label for="recharge-custom-amount" class="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="flex items-center gap-3 rounded-xl border border-gray-200 bg-white/80 px-4 focus-within:border-primary-400 focus-within:ring-2 focus-within:ring-primary-500/15 dark:border-dark-600 dark:bg-dark-950/40">
        <span class="shrink-0 text-sm font-medium text-gray-500 dark:text-dark-400">
          {{ currencySymbol(currency) }}
        </span>
        <input
          id="recharge-custom-amount"
          ref="customInput"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          :value="customText"
          :placeholder="placeholderText"
          class="min-w-0 flex-1 border-0 bg-transparent py-3 text-base font-medium text-gray-900 outline-none placeholder:font-normal placeholder:text-gray-400 focus:ring-0 dark:text-white"
          @input="handleInput"
        />
        <span class="shrink-0 text-xs font-medium text-gray-400 dark:text-dark-500">{{ currency }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { currencySymbol, formatPaymentAmount } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  currency?: string
  balanceMultiplier?: number
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  currency: 'USD',
  balanceMultiplier: 1,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')
const customInput = ref<HTMLInputElement | null>(null)
const customMode = ref(false)

function formatCreditedAmount(value: number) {
  return formatPaymentAmount(Math.round(value * props.balanceMultiplier * 100) / 100, 'USD')
}

async function focusCustomAmount() {
  customMode.value = true
  await nextTick()
  customInput.value?.focus()
}

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customMode.value = false
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
  customText.value = val
  customMode.value = true
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
    customMode.value = !filteredAmounts.value.includes(v)
  }
}, { immediate: true })
</script>

<style scoped>
.recharge-amount-card {
  @apply min-w-0 rounded-2xl border p-4 text-left shadow-glass-sm backdrop-blur-xl transition-colors duration-200 sm:p-5;
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-900;
}

.recharge-amount-card-idle {
  @apply border-white/80 bg-white/80 hover:border-primary-300 hover:bg-white dark:border-white/10 dark:bg-dark-900/70 dark:hover:border-primary-600 dark:hover:bg-dark-800;
}

.recharge-amount-card-selected {
  @apply border-primary-400 bg-primary-50/90 ring-1 ring-primary-400 dark:border-primary-500 dark:bg-primary-950/60 dark:ring-primary-500;
}
</style>
