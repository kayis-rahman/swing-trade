import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SignalTournamentPanel from './SignalTournamentPanel.vue'
import type { SignalSelection } from '../api/selections'

const row: SignalSelection = {
  id: 1,
  symbol: 'ADANIPORTS',
  selectionDate: '2026-09-18',
  winnerVariantId: 'pullback-v1',
  winnerVersion: 1,
  winnerConfidence: 0.75,
  candidates: [
    { variantId: 'breakout-v1', version: 1, signal: 'HOLD', confidence: 0.5, selected: false },
    { variantId: 'pullback-v1', version: 1, signal: 'BUY', confidence: 0.75, selected: true },
  ],
  reason: 'highest confidence 0.75 among 1 BUY(s)',
  status: 'PENDING',
  statusDetail: null,
}

describe('SignalTournamentPanel', () => {
  it('shows an empty message when there are no winners', () => {
    const wrapper = mount(SignalTournamentPanel, {
      props: { selections: [] },
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })
    expect(wrapper.text()).toContain('No variant produced a BUY')
  })

  it('renders the winner, every candidate chip and the selection status', () => {
    const wrapper = mount(SignalTournamentPanel, {
      props: { selections: [row] },
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })
    const text = wrapper.text()
    expect(text).toContain('ADANIPORTS')
    expect(text).toContain('pullback-v1 BUY 75%')
    expect(text).toContain('breakout-v1 HOLD 50%')
    expect(text).toContain('PENDING')
    expect(wrapper.find('[aria-label="selected"]').exists()).toBe(true)
  })
})
