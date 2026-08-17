import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ModelPlazaView from '../ModelPlazaView.vue'

const { listAccountModels, copyToClipboard, showError } = vi.hoisted(() => ({
  listAccountModels: vi.fn(),
  copyToClipboard: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelPlaza: {
      listAccountModels
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess: vi.fn()
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function sampleItems() {
  return {
    items: [
      {
        client_id: 'gpt-5.4',
        platform: 'openai',
        upstream_ids: ['gpt-5.4'],
        account_count: 1
      },
      {
        client_id: 'grok-4.5',
        platform: 'grok',
        upstream_ids: ['grok-4.5'],
        account_count: 1
      }
    ]
  }
}

function mountView() {
  return mount(ModelPlazaView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /></div>'
        },
        DataTable: {
          props: ['data', 'columns', 'loading'],
          template: `
            <div>
              <div v-if="!data || data.length === 0"><slot name="empty" /></div>
              <div v-for="row in data || []" :key="row.rowKey" class="row">
                <slot name="cell-client_id" :row="row" :value="row.client_id" />
                <slot name="cell-upstream_ids" :row="row" :value="row.upstream_ids" />
                <slot name="cell-platform" :row="row" :value="row.platform" />
                <slot name="cell-account_count" :row="row" :value="row.account_count" />
              </div>
            </div>
          `
        },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="opt in options" :key="String(opt.value)" :value="opt.value">{{ opt.label }}</option></select>'
        },
        Icon: true
      }
    }
  })
}

describe('Admin ModelPlazaView', () => {
  beforeEach(() => {
    listAccountModels.mockReset()
    copyToClipboard.mockReset()
    showError.mockReset()
  })

  it('renders aggregated account models and copies the client ID', async () => {
    listAccountModels.mockResolvedValue(sampleItems())
    const wrapper = mountView()
    await flushPromises()

    expect(listAccountModels).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('gpt-5.4')
    expect(wrapper.text()).toContain('grok-4.5')

    await wrapper.get('button[title="admin.modelPlaza.copyClientId"]').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith('gpt-5.4')
  })

  it('filters by search query', async () => {
    listAccountModels.mockResolvedValue(sampleItems())
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input').setValue('grok')
    expect(wrapper.text()).toContain('grok-4.5')
    expect(wrapper.text()).not.toContain('gpt-5.4')
  })

  it('shows empty copy when there are no enabled-account models', async () => {
    listAccountModels.mockResolvedValue({ items: [] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('admin.modelPlaza.empty')
  })
})
