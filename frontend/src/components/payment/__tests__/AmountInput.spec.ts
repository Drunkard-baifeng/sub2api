import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AmountInput from '../AmountInput.vue'
import { formatPaymentAmount } from '../currency'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params && 'amount' in params ? `${key} ${String(params.amount)}` : key,
  }),
}))
enableAutoUnmount(afterEach)

function mountInput(value: number | null = null) {
  return mount(AmountInput, { props: { modelValue: value } })
}

describe('recharge amount input', () => {
  it.each(['10abc', '10.555', '-10', '1e2'])('restores the accepted amount after rejecting %s', async (value) => {
    const wrapper = mountInput(10)
    const input = wrapper.get('input')
    await input.setValue(value)
    expect((input.element as HTMLInputElement).value).toBe('10')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('restores the last typed amount rather than a stale prop', async () => {
    const wrapper = mountInput()
    const input = wrapper.get('input')
    await input.setValue('12.50')
    await input.setValue('12.500')
    expect((input.element as HTMLInputElement).value).toBe('12.50')
    expect(wrapper.emitted('update:modelValue')).toEqual([[12.5]])
  })

  it('preserves decimal editing and allows clearing the amount', async () => {
    const wrapper = mountInput()
    const input = wrapper.get('input')
    for (const value of ['0', '0.', '0.5', '0.50', '']) await input.setValue(value)
    expect(wrapper.emitted('update:modelValue')).toEqual([[null], [null], [0.5], [0.5], [null]])
    expect((input.element as HTMLInputElement).value).toBe('')
  })

  it('filters preset cards by limits and shows the payment currency and configured credit amount', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [10, 50, 1000], min: 20, max: 100, currency: 'CNY', multiplier: 0.14 },
    })
    const cards = wrapper.findAll('[data-testid^="quick-amount-"][aria-pressed]')
    expect(cards).toHaveLength(1)
    expect(cards[0].text()).toContain('¥')
    expect(cards[0].text()).toContain('50')
    expect(cards[0].text()).toContain(formatPaymentAmount(7, 'USD'))
    expect(wrapper.get('input').attributes('placeholder')).toBe('20 - 100')
  })

  it('selects a card without losing custom decimal input', async () => {
    const wrapper = mountInput()
    await wrapper.get('[data-testid^="quick-amount-"][aria-pressed]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[10]])
    await wrapper.setProps({ modelValue: 10 })
    expect(wrapper.get('[data-testid^="quick-amount-"][aria-pressed]').attributes('aria-pressed')).toBe('true')
    await wrapper.get('input').setValue('12.50')
    expect(wrapper.get('[data-testid="recharge-custom-card"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([12.5])
  })

  it('updates the displayed currency and credit when the payment configuration changes', async () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50], currency: 'CNY', multiplier: 0.14 } })
    await wrapper.setProps({ currency: 'EUR', multiplier: 2 })
    expect(wrapper.get('[data-testid^="quick-amount-"][aria-pressed]').text()).toContain('€')
    expect(wrapper.get('[data-testid^="quick-amount-"][aria-pressed]').text()).toContain(formatPaymentAmount(100, 'USD'))
  })
})

describe('recharge bonus hints on quick amounts', () => {
  const tiers = [
    { min_amount: 100, bonus_percent: 20 },
    { min_amount: 500, bonus_percent: 30 },
  ]

  it('renders no badge or second line when no tiers are configured', () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50, 100, 500] } })
    expect(wrapper.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="quick-amount-credited"]').exists()).toBe(false)
  })

  it('bonus mode: price tag only on amounts that hit a tier, credited totals on every button', () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50, 100, 500], bonusTiers: tiers } })
    const below = wrapper.get('[data-testid="quick-amount-50"]')
    const first = wrapper.get('[data-testid="quick-amount-100"]')
    const second = wrapper.get('[data-testid="quick-amount-500"]')

    expect(below.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(below.get('[data-testid="quick-amount-credited"]').text()).toContain('$50.00')
    expect(first.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+20%')
    expect(first.get('[data-testid="quick-amount-credited"]').text()).toContain('$120.00')
    expect(second.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+30%')
    expect(second.get('[data-testid="quick-amount-credited"]').text()).toContain('$650.00')
  })

  it('bonus mode: matches tiers by the entered amount but credits by the multiplier', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [1000], bonusTiers: tiers, multiplier: 0.14 },
    })
    const button = wrapper.get('[data-testid="quick-amount-1000"]')
    expect(button.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('+30%')
    // 1000 × 0.14 = 140 base, +30% = 182
    expect(button.get('[data-testid="quick-amount-credited"]').text()).toContain('$182.00')
  })

  it('discount mode: tag reads N% OFF and the second line shows the discounted payment', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [50, 500], bonusTiers: tiers, bonusMode: 'discount', currency: 'USD' },
    })
    const below = wrapper.get('[data-testid="quick-amount-50"]')
    const hit = wrapper.get('[data-testid="quick-amount-500"]')
    expect(below.find('[data-testid="quick-amount-bonus-badge"]').exists()).toBe(false)
    expect(below.get('[data-testid="quick-amount-credited"]').text()).toContain('50.00')
    expect(hit.get('[data-testid="quick-amount-bonus-badge"]').text()).toBe('30% OFF')
    // 500 × (1 − 30%) = 350
    expect(hit.get('[data-testid="quick-amount-credited"]').text()).toContain('350.00')
  })

  it('keeps discounted presets within the payment channel limits and avoids misleading input bounds', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [50, 100, 200], min: 50, max: 90, bonusTiers: tiers, bonusMode: 'discount' },
    })
    expect(wrapper.find('[data-testid="quick-amount-50"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="quick-amount-100"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="quick-amount-200"]').exists()).toBe(false)
    expect(wrapper.get('input').attributes('placeholder')).toBe('payment.enterAmount')
  })
})
