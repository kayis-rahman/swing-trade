/* These inline component probes intentionally share one test file. */
/* eslint-disable vue/one-component-per-file */
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'

const appStateMocks = vi.hoisted(() => ({
  state: {
    healthStatus: 'checking',
    checking: false,
    healthError: null,
    lastHealthAttempt: null,
    lastHealthSuccess: null,
    bannerDismissed: false,
  },
  startHealthPolling: vi.fn().mockResolvedValue(undefined),
  stopHealthPolling: vi.fn(),
}))

vi.mock('./components/BackendDownBanner.vue', () => ({
  default: { name: 'BackendDownBanner' },
}))

vi.mock('./components/NotificationHost.vue', () => ({
  default: { name: 'NotificationHost' },
}))

vi.mock('./stores/appState', () => ({
  useAppStateStore: () => ({
    ...appStateMocks.state,
    startHealthPolling: appStateMocks.startHealthPolling,
    stopHealthPolling: appStateMocks.stopHealthPolling,
  }),
}))

vi.mock('./stores/settings', () => ({
  getSettings: () => ({ tradingConfig: { mode: 'paper' } }),
}))

describe('App route runtime boundary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps the route boundary mounted while updating its route reset scope', async () => {
    let boundaryInstance = 0
    const BoundaryProbe = defineComponent({
      name: 'RuntimeErrorBoundary',
      props: {
        resetKey: { type: String, default: '' },
        route: { type: String, default: '' },
      },
      setup(props, { slots }) {
        const instance = ++boundaryInstance
        return () =>
          h(
            'section',
            {
              'data-testid': 'runtime-boundary',
              'data-instance': instance,
              'data-reset-key': props.resetKey,
              'data-route': props.route,
            },
            [slots.default?.()]
          )
      },
    })
    let shellMountCount = 0
    const ShellProbe = defineComponent({
      setup() {
        shellMountCount += 1
        return () => h('div', { 'data-testid': 'shell-probe' }, 'Shell remains mounted')
      },
    })
    const FirstRoute = defineComponent({ render: () => h('p', 'First route') })
    const SecondRoute = defineComponent({ render: () => h('p', 'Second route') })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/first', component: FirstRoute },
        { path: '/second', component: SecondRoute },
      ],
    })
    await router.push('/first')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: {
          RuntimeErrorBoundary: BoundaryProbe,
          Sidebar: ShellProbe,
          Header: ShellProbe,
          BackendDownBanner: ShellProbe,
          NotificationHost: ShellProbe,
        },
      },
    })
    const firstBoundaryId = wrapper
      .get('[data-testid="runtime-boundary"]')
      .attributes('data-instance')
    expect(wrapper.get('[data-testid="runtime-boundary"]').attributes('data-reset-key')).toBe(
      '/first'
    )
    expect(shellMountCount).toBe(4)

    await router.push('/second')
    await nextTick()

    expect(wrapper.text()).toContain('Second route')
    expect(wrapper.get('[data-testid="runtime-boundary"]').attributes('data-instance')).toBe(
      firstBoundaryId
    )
    expect(wrapper.get('[data-testid="runtime-boundary"]').attributes('data-reset-key')).toBe(
      '/second'
    )
    expect(shellMountCount).toBe(4)

    wrapper.unmount()
  })
})
