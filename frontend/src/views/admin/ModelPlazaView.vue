<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-80">
            <input
              v-model.trim="searchQuery"
              type="text"
              class="input"
              :placeholder="t('admin.modelPlaza.searchPlaceholder')"
            />
          </div>
          <Select
            v-model="platformFilter"
            :options="platformOptions"
            class="w-40"
          />
          <div class="flex flex-1 items-center justify-end">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadItems"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredItems" :loading="loading" row-key="rowKey">
          <template #cell-client_id="{ row }">
            <button
              type="button"
              class="group/copy inline-flex max-w-full items-center gap-1 rounded-md text-left font-mono text-sm font-medium text-gray-900 transition-colors hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
              :title="t('admin.modelPlaza.copyClientId')"
              @click="copyId(row.client_id)"
            >
              <span class="truncate">{{ row.client_id }}</span>
              <Icon
                name="clipboard"
                size="xs"
                class="h-3.5 w-3.5 shrink-0 text-gray-400 opacity-0 transition-opacity group-hover/copy:opacity-100 group-focus-visible/copy:opacity-100 dark:text-dark-400"
              />
            </button>
          </template>

          <template #cell-upstream_ids="{ row }">
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="id in row.upstream_ids"
                :key="id"
                type="button"
                class="inline-flex max-w-full items-center rounded-md border border-gray-200 bg-gray-50 px-1.5 py-0.5 font-mono text-xs text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-700 dark:hover:text-primary-400"
                :title="t('admin.modelPlaza.copyUpstreamId')"
                @click="copyId(id)"
              >
                <span class="truncate">{{ id }}</span>
              </button>
              <span
                v-if="row.upstream_ids.length === 1 && row.upstream_ids[0] === row.client_id"
                class="self-center text-[11px] text-gray-400 dark:text-dark-500"
              >
                {{ t('admin.modelPlaza.sameAsClient') }}
              </span>
            </div>
          </template>

          <template #cell-platform="{ value }">
            <span
              :class="[
                'inline-flex items-center rounded-md border px-1.5 py-0.5 text-xs font-medium',
                platformBadgeClass(value)
              ]"
            >
              {{ platformLabel(value) }}
            </span>
          </template>

          <template #cell-account_count="{ value }">
            <span class="tabular-nums text-sm text-gray-700 dark:text-gray-200">{{ value }}</span>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-8">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-300 dark:text-dark-600" />
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.modelPlaza.empty') }}
              </p>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AccountModelPlazaItem } from '@/api/admin/modelPlaza'
import DataTable from '@/components/common/DataTable.vue'
import Select from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'

interface PlazaRow extends AccountModelPlazaItem {
  rowKey: string
}

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const items = ref<PlazaRow[]>([])
const searchQuery = ref('')
const platformFilter = ref('')

const columns = computed<Column[]>(() => [
  { key: 'client_id', label: t('admin.modelPlaza.clientId'), sortable: true },
  { key: 'upstream_ids', label: t('admin.modelPlaza.upstreamIds') },
  { key: 'platform', label: t('admin.modelPlaza.platform'), sortable: true },
  { key: 'account_count', label: t('admin.modelPlaza.accountCount'), sortable: true }
])

const platformOptions = computed(() => {
  const platforms = [...new Set(items.value.map((item) => item.platform))].sort()
  return [
    { value: '', label: t('admin.modelPlaza.allPlatforms') },
    ...platforms.map((platform) => ({
      value: platform,
      label: platformLabel(platform)
    }))
  ]
})

const filteredItems = computed(() => {
  const query = searchQuery.value.toLowerCase()
  return items.value.filter((item) => {
    if (platformFilter.value && item.platform !== platformFilter.value) {
      return false
    }
    if (!query) {
      return true
    }
    return (
      item.client_id.toLowerCase().includes(query) ||
      item.upstream_ids.some((id) => id.toLowerCase().includes(query))
    )
  })
})

function copyId(id: string) {
  void copyToClipboard(id)
}

async function loadItems() {
  loading.value = true
  try {
    const data = await adminAPI.modelPlaza.listAccountModels()
    items.value = (data.items ?? []).map((item) => ({
      ...item,
      rowKey: `${item.platform}:${item.client_id}`
    }))
  } catch {
    appStore.showError(t('admin.modelPlaza.loadFailed'))
    items.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadItems()
})
</script>
