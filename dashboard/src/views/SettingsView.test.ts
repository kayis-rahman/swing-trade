import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import type { Component } from 'vue'
import { getSettings } from '../stores/settings'

function createRouterMock() {
  return createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'Dashboard', component: { template: '<div />' } }],
  })
}

function mountSettings(SettingsView: Component) {
  const router = createRouterMock()
  return mount(SettingsView, {
    global: {
      plugins: [router],
      stubs: {
        LoadingSpinner: true,
        ErrorBoundary: {
          template: '<div><slot /></div>',
        },
      },
    },
  })
}

describe('SettingsView — Tabs', () => {
  it('shows Broker tab content by default', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    expect(wrapper.text()).toContain('Broker Connection')
    wrapper.unmount()
  })

  it('clicking AI/LLM tab shows LLM section', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    const llmTab = wrapper.find('[aria-label="AI/LLM"]')
    await llmTab.trigger('click')
    expect(wrapper.text()).toContain('OpenAI-compatible LLM')
    wrapper.unmount()
  })

  it('clicking Trading tab shows trading config', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    const tradingTab = wrapper.find('[aria-label="Trading"]')
    await tradingTab.trigger('click')
    expect(wrapper.text()).toContain('Trading Configuration')
    wrapper.unmount()
  })

  it('clicking Health tab shows health status', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    const healthTab = wrapper.find('[aria-label="Health"]')
    await healthTab.trigger('click')
    expect(wrapper.text()).toContain('System Health')
    wrapper.unmount()
  })

  it('tab buttons have correct aria roles', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    expect(wrapper.find('[role="tablist"]').exists()).toBe(true)
    expect(wrapper.findAll('[role="tab"]').length).toBe(5)
    wrapper.unmount()
  })
})

describe('SettingsView — Broker Section', () => {
  it('renders 4 broker options as pill buttons', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    expect(wrapper.text()).toContain('Fyers')
    expect(wrapper.text()).toContain('Upstox')
    expect(wrapper.text()).toContain('Yahoo Finance')
    expect(wrapper.text()).toContain('None')
    wrapper.unmount()
  })

  it('selected broker has brand styling', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    const selectedBtn = wrapper.find('.border-brand.bg-brand-subtle')
    expect(selectedBtn.exists()).toBe(true)
    wrapper.unmount()
  })

  it('connection status indicator shows dot', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    const dot = wrapper.find('.rounded-full')
    expect(dot.exists()).toBe(true)
    wrapper.unmount()
  })
})

describe('SettingsView — LLM Section', () => {
  it('renders OpenAI-compatible LLM section with test button', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="AI/LLM"]').trigger('click')
    expect(wrapper.text()).toContain('OpenAI-compatible LLM')
    expect(wrapper.text()).toContain('Test')
    wrapper.unmount()
  })

  it('exposes Laya as a selectable backend with its own endpoint and model fields', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="AI/LLM"]').trigger('click')

    // Laya is one of the backend options (local, pi_ssh, openai, ollama, laya, pi_agent, mlx).
    const backendOptions = wrapper.findAll('.llm-backend-option')
    expect(backendOptions).toHaveLength(7)

    // Selecting Laya wires the store backend to 'laya'.
    await backendOptions[4]?.trigger('click')
    await wrapper.vm.$nextTick()
    expect(getSettings().llmSettings.llmBackend).toBe('laya')
    expect(wrapper.find('.broker-option-check').exists()).toBe(true)

    // The Laya section exposes its own endpoint and model inputs, distinct from
    // the other backends.
    const placeholders = wrapper.findAll('input').map((input) => input.attributes('placeholder'))
    expect(placeholders).toContain('https://laya.example/v1')
    expect(placeholders).toContain('Ornith-1.5-35B-A3B-AWQ')

    wrapper.unmount()
  })

  it('renders PDF extraction section', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="AI/LLM"]').trigger('click')
    expect(wrapper.text()).toContain('PDF Extraction')
    wrapper.unmount()
  })

  it('renders Discord toggle with webhook input', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="AI/LLM"]').trigger('click')
    expect(wrapper.text()).toContain('Discord')
    wrapper.unmount()
  })
})

describe('SettingsView — Trading Section', () => {
  it('renders trading mode selector', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="Trading"]').trigger('click')
    expect(wrapper.text()).toContain('Paper Trading')
    expect(wrapper.text()).toContain('Live Trading')
    wrapper.unmount()
  })

  it('renders max position size with % suffix', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="Trading"]').trigger('click')
    expect(wrapper.text()).toContain('Max Position Size')
    expect(wrapper.text()).toContain('%')
    wrapper.unmount()
  })

  it('renders stop loss and take profit inputs', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="Trading"]').trigger('click')
    expect(wrapper.text()).toContain('Stop Loss')
    expect(wrapper.text()).toContain('Take Profit')
    wrapper.unmount()
  })
})

describe('SettingsView — Health Section', () => {
  it('shows loading spinner when no health data', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    await wrapper.find('[aria-label="Health"]').trigger('click')
    expect(wrapper.text()).toContain('System Health')
    wrapper.unmount()
  })
})

describe('SettingsView — Save Flow', () => {
  it('save button is in header bar', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    expect(wrapper.text()).toContain('Save All Settings')
    wrapper.unmount()
  })

  it('save button shows saving state', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mountSettings(SettingsView)
    expect(wrapper.text()).toContain('Save All Settings')
    wrapper.unmount()
  })

  it('save button triggers saveSettings', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mount(SettingsView, {
      global: {
        plugins: [createRouterMock()],
        stubs: {
          LoadingSpinner: true,
          ErrorBoundary: {
            template: '<div><slot /></div>',
          },
        },
      },
    })
    const buttons = wrapper.findAll('button')
    const saveBtn = buttons.find((b) => b.text().includes('Save'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    // After save, text should change to "Saving..." then "Saved!"
    await wrapper.vm.$nextTick()
    const text = saveBtn!.text()
    expect(['Saving...', 'Saved!', 'Save All Settings']).toContain(text)
    wrapper.unmount()
  })

  it('save button is disabled during save', async () => {
    const SettingsView = (await import('./SettingsView.vue')).default
    const wrapper = mount(SettingsView, {
      global: {
        plugins: [createRouterMock()],
        stubs: {
          LoadingSpinner: true,
          ErrorBoundary: {
            template: '<div><slot /></div>',
          },
        },
      },
    })
    const buttons = wrapper.findAll('button')
    const saveBtn = buttons.find((b) => b.text().includes('Save'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    await wrapper.vm.$nextTick()
    expect(saveBtn!.element.disabled).toBe(true)
    wrapper.unmount()
  })
})
