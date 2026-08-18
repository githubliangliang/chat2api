import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

// --- i18n messages (only what UsageFilters needs) ---
const messages: Record<string, string> = {
  'admin.usage.userDeletedBadge': 'deleted',
  'admin.usage.userFilter': 'User',
  'admin.usage.searchUserPlaceholder': 'Search user...',
  'usage.apiKeyFilter': 'API Key',
  'admin.usage.searchApiKeyPlaceholder': 'Search API key...',
  'usage.model': 'Model',
  'admin.usage.allModels': 'All Models',
  'admin.usage.account': 'Account',
  'admin.usage.searchAccountPlaceholder': 'Search account...',
  'usage.type': 'Type',
  'admin.usage.allTypes': 'All Types',
  'usage.ws': 'WS',
  'usage.stream': 'Stream',
  'usage.sync': 'Sync',
  'admin.usage.billingType': 'Billing Type',
  'admin.usage.allBillingTypes': 'All Billing Types',
  'admin.usage.billingTypeBalance': 'Balance',
  'admin.usage.billingTypeSubscription': 'Subscription',
  'admin.usage.billingMode': 'Billing Mode',
  'admin.usage.allBillingModes': 'All Billing Modes',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per Request',
  'admin.usage.billingModeImage': 'Image',
	'admin.usage.upstreamModelAudit': 'Upstream model audit',
	'admin.usage.allUpstreamModelAudit': 'All response model states',
	'admin.usage.upstreamModelMismatchOnly': 'Mismatched only',
	'admin.usage.upstreamModelMatchedOnly': 'Matched only',
  'admin.usage.group': 'Group',
  'admin.usage.allGroups': 'All Groups',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
  'admin.usage.cleanup.button': 'Cleanup',
  'admin.ops.errorLog.cleanupAll': 'Cleanup errors',
  'admin.usage.moreFilters': 'More filters',
  'usage.exportExcel': 'Export',
}

// Mock vue-i18n
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

// Mock the admin API module — we control searchUsers return value per test
const mockSearchUsers = vi.fn()
const mockSearchApiKeys = vi.fn().mockResolvedValue([])
const mockGroupsList = vi.fn().mockResolvedValue({ items: [] })
const mockGetModelStats = vi.fn().mockResolvedValue({ models: [] })
const mockAccountsList = vi.fn().mockResolvedValue({ items: [] })

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: (...args: any[]) => mockSearchUsers(...args),
      searchApiKeys: (...args: any[]) => mockSearchApiKeys(...args),
    },
    groups: { list: (...args: any[]) => mockGroupsList(...args) },
    dashboard: { getModelStats: (...args: any[]) => mockGetModelStats(...args) },
    accounts: { list: (...args: any[]) => mockAccountsList(...args) },
  },
}))

// Default props helper
const defaultFilters = () => ({
  user_id: undefined,
  api_key_id: undefined,
  account_id: undefined,
  model: null,
  request_type: null,
  billing_type: null,
  billing_mode: null,
	upstream_model_mismatch: null,
  group_id: null,
  start_date: '',
  end_date: '',
})

function mountFilters(filters = defaultFilters()) {
  return mount(UsageFilters, {
    props: {
      modelValue: filters,
      exporting: false,
      startDate: '2026-05-01',
      endDate: '2026-05-28',
      showActions: false,
      modelOptions: [],
    },
    global: {
      stubs: {
        Select: true,
        Teleport: true,
      },
    },
  })
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })

  return { promise, resolve, reject }
}

describe('UsageFilters — user search dropdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockSearchUsers.mockReset()
    mockSearchApiKeys.mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('(a) labels deleted users with the i18n badge and (b) sorts active users before deleted ones, (c) selection sets user_id', async () => {
    // Arrange: mock returns deleted FIRST (proves sorting re-orders to active-first)
    mockSearchUsers.mockResolvedValue([
      { id: 2, email: 'gone@test.com', deleted: true },
      { id: 1, email: 'active@test.com', deleted: false },
    ])

    const wrapper = mountFilters()
    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')

    // Trigger focus (sets showUserDropdown = true) then input (fires debounceUserSearch)
    const input = wrapper.get('[data-test="advanced-filter-user"] input')
    await input.trigger('focus')
    await input.setValue('test')
    await input.trigger('input')

    // Advance debounce timer (300ms) then flush the resolved promise
    vi.advanceTimersByTime(300)
    await flushPromises()

    // --- (b) Sort: active user should appear BEFORE deleted user ---
    // Check the underlying component state via rendered DOM order
    const buttons = wrapper.findAll('.usage-filter-dropdown button[type="button"]')
    const emailTexts = buttons.map((b) => b.text())

    // active@test.com should be listed first
    const activeIdx = emailTexts.findIndex((t) => t.includes('active@test.com'))
    const deletedIdx = emailTexts.findIndex((t) => t.includes('gone@test.com'))
    expect(activeIdx).toBeGreaterThanOrEqual(0)
    expect(deletedIdx).toBeGreaterThanOrEqual(0)
    expect(activeIdx).toBeLessThan(deletedIdx)

    // --- (a) Label: deleted user's button shows the badge text ---
    const deletedButton = buttons[deletedIdx]
    expect(deletedButton.text()).toContain('deleted')

    // active user's button does NOT show the badge text
    const activeButton = buttons[activeIdx]
    expect(activeButton.text()).not.toContain('deleted')

    // --- (c) Selection: clicking active user button sets filters.user_id ---
    await activeButton.trigger('click')
    await flushPromises()

    // The component emits 'update:modelValue' or modifies filters.user_id via toRef
    // selectUser sets filters.value.user_id = u.id and emits 'change'
    const changeEmits = wrapper.emitted('change')
    expect(changeEmits).toBeTruthy()
    expect(changeEmits!.length).toBeGreaterThan(0)

    // Also confirm user_id was set by checking the emitted change came through
    // (the component uses toRef so modelValue is mutated in place and 'change' is emitted)
    expect(wrapper.props('modelValue').user_id).toBe(1)
  })

  it('keeps results from the latest user search when responses arrive out of order', async () => {
    const firstSearch = deferred<Array<{ id: number; email: string; deleted: boolean }>>()
    const secondSearch = deferred<Array<{ id: number; email: string; deleted: boolean }>>()
    mockSearchUsers
      .mockImplementationOnce(() => firstSearch.promise)
      .mockImplementationOnce(() => secondSearch.promise)

    const wrapper = mountFilters()
    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')
    const input = wrapper.get('[data-test="advanced-filter-user"] input')
    await input.trigger('focus')

    await input.setValue('a')
    vi.advanceTimersByTime(300)
    await flushPromises()

    await input.setValue('ab')
    vi.advanceTimersByTime(300)
    await flushPromises()

    secondSearch.resolve([{ id: 2, email: 'ab@test.com', deleted: false }])
    await flushPromises()
    expect(wrapper.text()).toContain('ab@test.com')

    firstSearch.resolve([{ id: 1, email: 'a@test.com', deleted: false }])
    await flushPromises()
    expect(wrapper.text()).toContain('ab@test.com')
    expect(wrapper.text()).not.toContain('a@test.com')
  })

  it('does not restore stale user results after the search is cleared', async () => {
    const pendingSearch = deferred<Array<{ id: number; email: string; deleted: boolean }>>()
    mockSearchUsers.mockImplementationOnce(() => pendingSearch.promise)

    const wrapper = mountFilters()
    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')
    const input = wrapper.get('[data-test="advanced-filter-user"] input')
    await input.trigger('focus')

    await input.setValue('stale')
    vi.advanceTimersByTime(300)
    await flushPromises()

    await input.setValue('')
    vi.advanceTimersByTime(300)
    await flushPromises()

    pendingSearch.resolve([{ id: 3, email: 'stale@test.com', deleted: false }])
    await flushPromises()
    expect(wrapper.text()).not.toContain('stale@test.com')
  })
})

describe('UsageFilters error cleanup action', () => {
  it('shows a cleanup button for errors and emits cleanup', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: true,
        mode: 'errors',
      },
      global: { stubs: { Select: true, Teleport: true } },
    })

    const button = wrapper.findAll('button').find((item) => item.text() === 'Cleanup errors')
    expect(button).toBeDefined()
    await button!.trigger('click')
    expect(wrapper.emitted('cleanup')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('Export')
  })
})

describe('UsageFilters compact filter layout', () => {
  const mountMode = (mode: 'usage' | 'errors' | 'ranking', modelValue = defaultFilters()) => mount(UsageFilters, {
    props: {
      modelValue,
      exporting: false,
      startDate: '2026-05-01',
      endDate: '2026-05-28',
      showActions: false,
      mode,
    },
    global: { stubs: { Select: true, Teleport: true, Icon: true } },
  })

  it.each([
    ['usage', false, false],
    ['errors', true, false],
    ['ranking', false, true],
  ] as const)('keeps model and group visible while %s-specific filters expand from the dropdown', async (mode, hasErrorFilters, hasRankingFilters) => {
    const wrapper = mountMode(mode)

    expect(wrapper.find('[data-test="primary-filter-model"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="primary-filter-group"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="advanced-filter-user"]').exists()).toBe(false)

    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')

    expect(wrapper.find('[data-test="advanced-filter-user"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="advanced-filter-api-key"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="advanced-filter-account"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="advanced-filter-error-type"]').exists()).toBe(hasErrorFilters)
    expect(wrapper.find('[data-test="advanced-filter-request-type"]').exists()).toBe(hasRankingFilters)
  })

  it('shows the number of active advanced filters on the dropdown trigger', () => {
    const filters = defaultFilters()
    filters.user_id = 12
    filters.account_id = 7
    const wrapper = mountMode('usage', filters)

    expect(wrapper.get('[data-test="more-filter-count"]').text()).toBe('2')
  })

  it('aligns the dropdown to the filter row to avoid viewport overflow', async () => {
    const wrapper = mountMode('usage')

    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')

    expect(wrapper.get('[data-test="usage-filter-row"]').classes()).toContain('relative')
    const panelClasses = wrapper.get('#usage-advanced-filters').classes()
    expect(panelClasses).toContain('left-0')
    expect(panelClasses).not.toContain('right-0')
  })

  it('places filters and actions on one row only at the wide-screen breakpoint', () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: true,
      },
      global: { stubs: { Select: true, Teleport: true, Icon: true } },
    })

    const toolbar = wrapper.get('[data-test="usage-filter-row"]').element.parentElement
    expect(toolbar?.classList.contains('2xl:flex-row')).toBe(true)

    const actions = wrapper.find('[data-test="usage-filter-actions"]')
    expect(actions.exists()).toBe(true)
    expect(actions.classes()).toContain('2xl:shrink-0')
  })

})

describe('UsageFilters — model options come from prop (no dup request)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetModelStats.mockClear()
    mockGroupsList.mockClear()
  })
  afterEach(() => { vi.useRealTimers() })

  it('does not call dashboard.getModelStats on mount and renders model options from prop', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: ['claude-3', 'gpt-4o'],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
    await flushPromises()

    expect(mockGetModelStats).not.toHaveBeenCalled()

    const opts = (wrapper.vm as any).modelOptions as Array<{ value: string | null; label: string }>
    expect(opts.map((o) => o.value)).toEqual([null, 'claude-3', 'gpt-4o'])
  })
})

describe('UsageFilters — usage tab hides billing/type filters', () => {
  function mountWithMode(mode: 'usage' | 'errors' | 'ranking') {
    return mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: [],
        mode,
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
  }

  it('hides type, billing type, billing mode, and upstream model audit on usage details', async () => {
    const wrapper = mountWithMode('usage')
    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')
    const text = wrapper.text()
    expect(text).not.toContain('Type')
    expect(text).not.toContain('Billing Type')
    expect(text).not.toContain('Billing Mode')
    expect(text).not.toContain('Upstream model audit')
  })

  it('keeps type and billing type on the ranking tab', async () => {
    const wrapper = mountWithMode('ranking')
    await wrapper.get('[data-test="usage-more-filters"]').trigger('click')
    const text = wrapper.text()
    expect(text).toContain('Type')
    expect(text).toContain('Billing Type')
    expect(text).not.toContain('Billing Mode')
    expect(text).not.toContain('Upstream model audit')
  })
})
