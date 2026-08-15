import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'

import SetupWizardView from '../SetupWizardView.vue'
import { testDatabase, testRedis, install } from '@/api/setup'

vi.mock('@/api/setup', () => ({
  testDatabase: vi.fn().mockResolvedValue(undefined),
  testRedis: vi.fn().mockResolvedValue(undefined),
  install: vi.fn().mockResolvedValue({ message: 'ok', restart: true })
}))

vi.mock('@/api/client', () => ({
  buildGatewayUrl: (path: string) => path
}))

const messages: Record<string, string> = {
  'common.next': 'Next',
  'common.back': 'Back',
  'setup.status.testConnection': 'Test Connection',
  'setup.status.testing': 'Testing...',
  'setup.status.success': 'Connection Successful',
  'setup.status.completeInstallation': 'Complete Installation',
  'setup.status.installing': 'Installing...',
  'setup.redis.useExternal': 'Use external Redis',
  'setup.redis.embedded': 'Embedded Redis (single-node)'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
      locale: { value: 'en' }
    })
  }
})

function mountWizard() {
  return mount(SetupWizardView, {
    global: {
      stubs: { Icon: true }
    }
  })
}

function button(wrapper: VueWrapper, label: string) {
  const found = wrapper.findAll('button').find((btn) => btn.text().includes(label))
  if (!found) {
    throw new Error(`button "${label}" not found`)
  }
  return found
}

/** Completes step 1 (database) and lands on the Redis step. */
async function gotoRedisStep(wrapper: VueWrapper) {
  await button(wrapper, 'Test Connection').trigger('click')
  await flushPromises()
  await button(wrapper, 'Next').trigger('click')
  await flushPromises()
}

describe('SetupWizardView redis step', () => {
  beforeEach(() => {
    vi.mocked(testDatabase).mockClear()
    vi.mocked(testRedis).mockClear()
    vi.mocked(install).mockClear()
    // waitForServiceRestart polls this after install; never resolve so the test stays put.
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))
  })

  it('defaults to embedded redis and proceeds without a connection test', async () => {
    const wrapper = mountWizard()
    await gotoRedisStep(wrapper)

    expect(wrapper.text()).toContain('Use external Redis')
    // Connection fields and the test button only exist for external Redis.
    expect(wrapper.find('input[type="password"]').exists()).toBe(false)
    expect(button(wrapper, 'Next').attributes('disabled')).toBeUndefined()

    await button(wrapper, 'Next').trigger('click')
    await flushPromises()

    expect(vi.mocked(testRedis)).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('sends redis.enabled=false to install when embedded redis is kept', async () => {
    const wrapper = mountWizard()
    await gotoRedisStep(wrapper)
    await button(wrapper, 'Next').trigger('click')

    // Admin step
    await wrapper.find('input[type="email"]').setValue('admin@example.com')
    const passwords = wrapper.findAll('input[type="password"]')
    await passwords[0].setValue('supersecret')
    await passwords[1].setValue('supersecret')
    await button(wrapper, 'Next').trigger('click')

    expect(wrapper.text()).toContain('Embedded Redis (single-node)')

    await button(wrapper, 'Complete Installation').trigger('click')
    await flushPromises()

    expect(vi.mocked(install)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(install).mock.calls[0][0].redis).toMatchObject({
      enabled: false,
      host: 'localhost'
    })
    wrapper.unmount()
  })

  it('requires a successful connection test when external redis is enabled', async () => {
    const wrapper = mountWizard()
    await gotoRedisStep(wrapper)

    await wrapper.find('[data-test="redis-enabled-toggle"]').trigger('click')
    await flushPromises()

    expect(button(wrapper, 'Next').attributes('disabled')).toBeDefined()

    await button(wrapper, 'Test Connection').trigger('click')
    await flushPromises()

    expect(vi.mocked(testRedis)).toHaveBeenCalledWith(expect.objectContaining({ enabled: true }))
    expect(button(wrapper, 'Next').attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('invalidates an earlier connection test when switching back to embedded redis', async () => {
    const wrapper = mountWizard()
    await gotoRedisStep(wrapper)

    await wrapper.find('[data-test="redis-enabled-toggle"]').trigger('click')
    await button(wrapper, 'Test Connection').trigger('click')
    await flushPromises()

    // Off (embedded) → on again: the previous success must not carry over.
    await wrapper.find('[data-test="redis-enabled-toggle"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-test="redis-enabled-toggle"]').trigger('click')
    await flushPromises()

    expect(button(wrapper, 'Next').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
