import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'

import UsageView from '../UsageView.vue'

const { list, exportList, getStats, getById, getModelStats, listErrorLogs, routeQuery, aoaToSheet, sheetAddAoa, saveAs, xlsxWrite } = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    list: vi.fn(),
    exportList: vi.fn(),
    getStats: vi.fn(),
    getById: vi.fn(),
    getModelStats: vi.fn(),
    listErrorLogs: vi.fn(),
    routeQuery: {} as Record<string, string>,
    aoaToSheet: vi.fn(() => ({})),
    sheetAddAoa: vi.fn(),
    saveAs: vi.fn(),
    xlsxWrite: vi.fn(() => new Uint8Array([1, 2, 3])),
  }
})

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time Range',
  'admin.usage.failedToLoadUser': 'Failed to load user',
  'admin.usage.requestId': 'Request ID',
  'usage.requestedModel': 'Requested model',
  'usage.sentUpstreamModel': 'Sent upstream model',
  'usage.upstreamResponseModel': 'Upstream response model',
  'usage.upstreamModelMismatch': 'Upstream model mismatch',
  'common.yes': 'Yes',
  'common.no': 'No',
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      list,
      getStats,
    },
    dashboard: {
      getModelStats,
    },
    users: {
      getById,
    },
  },
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    list: exportList,
  },
}))

vi.mock('file-saver', () => ({ saveAs }))

vi.mock('xlsx', () => ({
  utils: {
    aoa_to_sheet: aoaToSheet,
    sheet_add_aoa: sheetAddAoa,
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
  },
  write: xlsxWrite,
}))

vi.mock('@/api/admin/ops', () => ({
  listErrorLogs,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/utils/format', () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? '-',
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery
  })
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = defineComponent({
  setup(_, { expose }) {
    const userKeyword = ref('')
    let userSearchRevision = 0
    const setUserKeyword = (email: string) => {
      userSearchRevision += 1
      userKeyword.value = email
    }
    expose({
      getUserSearchRevision: () => userSearchRevision,
      setUserKeyword,
      simulateUserInput: setUserKeyword,
    })
    return { userKeyword }
  },
  template: '<div><span data-test="user-filter-label">{{ userKeyword }}</span><slot name="after-reset" /></div>',
})
const UsageTableStub = {
  props: ['columns'],
  emits: ['userClick'],
  template: '<div data-test="usage-table"><button class="user-click" @click="$emit(\'userClick\', 2)">user</button></div>',
}
const UserTokenRankingStub = {
  emits: ['select-user'],
  template: '<div data-test="ranking"><button class="pick-user" @click="$emit(\'select-user\', 5, \'rank@test.com\')">pick</button></div>',
}

const defaultStubs = {
  AppLayout: AppLayoutStub,
  UsageStatsCards: true,
  UsageFilters: UsageFiltersStub,
  UsageTable: true,
  UsageExportProgress: true,
  UsageCleanupDialog: true,
  UserBalanceHistoryModal: true,
  Pagination: true,
  DateRangePicker: true,
  Icon: true,
  UserTokenRanking: true,
}

const mountRouteFilteredUsageView = () => mount(UsageView, {
  global: { stubs: defaultStubs },
})

const emptyStats = {
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  average_duration_ms: 0,
}

describe('admin UsageView route filters', () => {
  beforeEach(() => {
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getById.mockReset()
  })

  afterEach(() => {
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
  })

  it('shows the routed user while applying user_id to usage requests', async () => {
    routeQuery.user_id = '42'
    getById.mockResolvedValue({ id: 42, email: 'route-user@test.com' })

    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(42, true)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 42 }), expect.anything())
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('route-user@test.com')
  })

  it('does not apply a stale routed user label after user_id changes', async () => {
    routeQuery.user_id = '42'
    let resolveLookup!: (user: { id: number; email: string }) => void
    getById.mockReturnValue(new Promise((resolve) => { resolveLookup = resolve }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.vm as any).filters.user_id = 84
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).setUserKeyword('current-user@test.com')

    resolveLookup({ id: 42, email: 'stale-user@test.com' })
    await flushPromises()

    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('current-user@test.com')
  })

  it('does not overwrite newer user input when the routed user lookup succeeds', async () => {
    routeQuery.user_id = '42'
    let resolveLookup!: (user: { id: number; email: string }) => void
    getById.mockReturnValue(new Promise((resolve) => { resolveLookup = resolve }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).simulateUserInput('new-search@test.com')

    resolveLookup({ id: 42, email: 'route-user@test.com' })
    await flushPromises()

    expect((wrapper.vm as any).filters.user_id).toBe(42)
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('new-search@test.com')
  })

  it('does not overwrite newer user input when the routed user lookup fails', async () => {
    routeQuery.user_id = '42'
    let rejectLookup!: (error: Error) => void
    getById.mockReturnValue(new Promise((_, reject) => { rejectLookup = reject }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).simulateUserInput('new-search@test.com')

    rejectLookup(new Error('lookup failed'))
    await flushPromises()

    expect((wrapper.vm as any).filters.user_id).toBe(42)
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('new-search@test.com')
  })

  it('shows the routed user ID when its label lookup fails', async () => {
    routeQuery.user_id = '42'
    getById.mockRejectedValue(new Error('lookup failed'))

    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 42 }), expect.anything())
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('42')
  })
})

describe('admin UsageView request ID column visibility', () => {
  beforeEach(() => {
    vi.mocked(localStorage.getItem).mockReset().mockReturnValue(null)
    vi.mocked(localStorage.setItem).mockReset()
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
  })

  it('keeps request ID hidden by default and allows enabling it from column settings', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          ...defaultStubs,
          UsageTable: UsageTableStub,
        },
      },
    })
    await wrapper.vm.$nextTick()

    const usageTable = wrapper.findComponent(UsageTableStub)
    expect(usageTable.props('columns')).not.toEqual(
      expect.arrayContaining([expect.objectContaining({ key: 'request_id' })]),
    )

    await wrapper.get('button[title="admin.users.columnSettings"]').trigger('click')
    const requestIdToggle = wrapper.findAll('button').find((button) => button.text() === 'Request ID')
    expect(requestIdToggle).toBeDefined()
    await requestIdToggle!.trigger('click')

    expect(usageTable.props('columns')).toEqual(
      expect.arrayContaining([expect.objectContaining({ key: 'request_id', label: 'Request ID' })]),
    )
    expect(localStorage.setItem).toHaveBeenCalledWith(
      'usage-hidden-columns-version',
      'request-id-hidden-by-default',
    )
  })
})

describe('admin UsageView handleUserClick', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getById.mockReset()
  })

  it('opens user via include_deleted when clicking a usage row user', async () => {
    getById.mockResolvedValue({ id: 2, email: 'd@test.com', deleted_at: '2026-05-28T00:00:00Z' })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          ...defaultStubs,
          UsageTable: UsageTableStub,
        },
      },
    })

    await flushPromises()

    await wrapper.find('[data-test="usage-table"] .user-click').trigger('click')
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(2, true)
  })
})

describe('admin UsageView errors tab filter forwarding', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    listErrorLogs.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  it('forwards model/account_id/group_id to listErrorLogs on the errors tab', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        ...defaultStubs,
        OpsErrorLogTable: true,
        OpsErrorDetailModal: true,
      } },
    })
    await flushPromises()

    const vm = wrapper.vm as any
    vm.filters.model = 'gpt-5.3-codex'
    vm.filters.account_id = 7
    vm.filters.group_id = 3
    await flushPromises()

    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    await tabs[1].trigger('click')
    await flushPromises()

    expect(listErrorLogs).toHaveBeenCalledWith(expect.objectContaining({
      view: 'all',
      model: 'gpt-5.3-codex',
      account_id: 7,
      group_id: 3,
    }))
  })
})

describe('admin UsageView ranking tab', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
  })

  it('mounts ranking lazily and drill-down sets user filter then jumps back to usage tab', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        ...defaultStubs,
        UserTokenRanking: UserTokenRankingStub,
        OpsErrorLogTable: true,
        OpsErrorDetailModal: true,
      } },
    })
    await flushPromises()

    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(false)

    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    expect(tabs).toHaveLength(3)
    await tabs[2].trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(true)

    list.mockClear()
    await wrapper.find('[data-test="ranking"] .pick-user').trigger('click')
    await flushPromises()

    expect((wrapper.vm as any).activeTab).toBe('usage')
    expect((wrapper.vm as any).filters.user_id).toBe(5)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 5 }), expect.anything())
  })
})

describe('admin UsageView model audit export', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    exportList.mockReset().mockResolvedValue({
      items: [{
        id: 1,
        created_at: '2026-08-04T00:00:00Z',
        model: 'gpt-5.6-sol',
        upstream_model: 'gpt-5.5',
        upstream_response_model: 'gpt-5.4',
        upstream_model_mismatch: true,
        request_type: 'sync',
        input_tokens: 1,
        output_tokens: 1,
        cache_read_tokens: 0,
        cache_creation_tokens: 0,
        duration_ms: 10,
      }],
      total: 1,
      pages: 1,
    })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    aoaToSheet.mockClear()
    sheetAddAoa.mockClear()
    saveAs.mockClear()
    xlsxWrite.mockClear()
  })

  it('exports requested, sent, response, and mismatch as separate admin columns', async () => {
    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    await (wrapper.vm as any).exportToExcel()
    await flushPromises()

    const headers = aoaToSheet.mock.calls[0][0][0]
    expect(headers.slice(4, 8)).toEqual([
      'Requested model',
      'Sent upstream model',
      'Upstream response model',
      'Upstream model mismatch',
    ])
    const row = sheetAddAoa.mock.calls[0][1][0]
    expect(row.slice(4, 8)).toEqual(['gpt-5.6-sol', 'gpt-5.5', 'gpt-5.4', 'Yes'])
    expect(saveAs).toHaveBeenCalledTimes(1)
  })
})

const UsageFiltersModelOptionsStub = defineComponent({
  props: {
    modelOptions: { type: Array, default: () => [] },
  },
  template: '<div data-test="model-options">{{ (modelOptions || []).join(",") }}</div>',
})

describe('admin UsageView model filter options', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
  })

  const mountWithModelFilter = () => mount(UsageView, {
    global: { stubs: {
      ...defaultStubs,
      UsageFilters: UsageFiltersModelOptionsStub,
    } },
  })

  it('fills the model dropdown from usage logs when dashboard model stats are empty', async () => {
    list.mockResolvedValue({
      items: [
        { id: 1, model: 'claude-sonnet-4-6' },
        { id: 2, model: 'gpt-5.4' },
        { id: 3, model: 'claude-sonnet-4-6' },
      ],
      total: 3,
      pages: 1,
    })
    getModelStats.mockResolvedValue({ models: [] })

    const wrapper = mountWithModelFilter()
    await flushPromises()

    expect(wrapper.find('[data-test="model-options"]').text().split(',').filter(Boolean).sort()).toEqual([
      'claude-sonnet-4-6',
      'gpt-5.4',
    ])
  })

  it('fills the model dropdown from requested model stats when logs have no models', async () => {
    list.mockResolvedValue({ items: [{ id: 1 }], total: 1, pages: 1 })
    getModelStats.mockResolvedValue({
      models: [
        { model: 'claude-opus-4-6', total_tokens: 10 },
        { model: 'gpt-5.4', total_tokens: 4 },
      ],
    })

    const wrapper = mountWithModelFilter()
    await flushPromises()

    expect(wrapper.find('[data-test="model-options"]').text().split(',').filter(Boolean).sort()).toEqual([
      'claude-opus-4-6',
      'gpt-5.4',
    ])
  })
})

describe('admin UsageView chart removal', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
  })

  it('does not render distribution or trend charts', async () => {
    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ModelDistributionChart' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'GroupDistributionChart' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'EndpointDistributionChart' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'TokenUsageTrend' }).exists()).toBe(false)
  })
})
