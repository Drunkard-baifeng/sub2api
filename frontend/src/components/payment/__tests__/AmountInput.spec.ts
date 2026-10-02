import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AmountInput from '../AmountInput.vue'
import { formatPaymentAmount } from '../currency'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
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
      props: { modelValue: null, amounts: [10, 50, 1000], min: 20, max: 100, currency: 'CNY', balanceMultiplier: 0.14 },
    })
    const cards = wrapper.findAll('[data-testid="recharge-amount-card"]')
    expect(cards).toHaveLength(1)
    expect(cards[0].text()).toContain('¥')
    expect(cards[0].text()).toContain('50')
    expect(cards[0].text()).toContain(formatPaymentAmount(7, 'USD'))
    expect(wrapper.get('input').attributes('placeholder')).toBe('20 - 100')
  })

  it('selects a card without losing custom decimal input', async () => {
    const wrapper = mountInput()
    await wrapper.get('[data-testid="recharge-amount-card"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[10]])
    await wrapper.setProps({ modelValue: 10 })
    expect(wrapper.get('[data-testid="recharge-amount-card"]').attributes('aria-pressed')).toBe('true')
    await wrapper.get('input').setValue('12.50')
    expect(wrapper.get('[data-testid="recharge-custom-card"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([12.5])
  })

  it('updates the displayed currency and credit when the payment configuration changes', async () => {
    const wrapper = mount(AmountInput, { props: { modelValue: null, amounts: [50], currency: 'CNY', balanceMultiplier: 0.14 } })
    await wrapper.setProps({ currency: 'EUR', balanceMultiplier: 2 })
    expect(wrapper.get('[data-testid="recharge-amount-card"]').text()).toContain('€')
    expect(wrapper.get('[data-testid="recharge-amount-card"]').text()).toContain(formatPaymentAmount(100, 'USD'))
  })
})
