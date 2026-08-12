<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6 p-4 sm:p-6">
      <!-- Header -->
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.menuManagement.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.menuManagement.description') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading || saving"
            @click="load"
          >
            {{ t('common.refresh') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="loading || saving || !dirty"
            @click="save"
          >
            <span v-if="saving">{{ t('common.saving') }}</span>
            <span v-else>{{ t('common.save') }}</span>
          </button>
        </div>
      </div>

      <!-- Tabs -->
      <div class="flex gap-1 rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition"
          :class="
            activeTab === tab.id
              ? 'bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-400'
              : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'
          "
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
          <span
            v-if="tab.badge != null"
            class="ml-1 rounded-full bg-gray-200 px-1.5 py-0.5 text-xs dark:bg-dark-600"
          >
            {{ tab.badge }}
          </span>
        </button>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <LoadingSpinner />
      </div>

      <template v-else>
        <!-- Built-in menus -->
        <div v-show="activeTab === 'builtin'" class="space-y-6">
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.menuManagement.userMenus') }}
                  </h2>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.menuManagement.userMenusHint') }}
                  </p>
                </div>
                <div class="flex gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="setScopeVisible('user', true)">
                    {{ t('admin.menuManagement.showAll') }}
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" @click="setScopeVisible('user', false)">
                    {{ t('admin.menuManagement.hideAll') }}
                  </button>
                </div>
              </div>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-dark-700">
              <li
                v-for="item in USER_BUILTIN_MENUS"
                :key="item.path"
                class="flex items-center justify-between gap-4 px-5 py-3"
              >
                <div class="min-w-0">
                  <div class="font-medium text-gray-900 dark:text-white">
                    {{ t(item.labelKey) }}
                  </div>
                  <div class="truncate font-mono text-xs text-gray-400">{{ item.path }}</div>
                </div>
                <div class="flex items-center gap-3">
                  <span
                    class="text-xs"
                    :class="isVisible(item.path) ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400'"
                  >
                    {{ isVisible(item.path) ? t('admin.menuManagement.visible') : t('admin.menuManagement.hidden') }}
                  </span>
                  <Toggle :model-value="isVisible(item.path)" @update:model-value="setVisible(item.path, $event)" />
                </div>
              </li>
            </ul>
          </div>

          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.menuManagement.adminMenus') }}
                  </h2>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.menuManagement.adminMenusHint') }}
                  </p>
                </div>
                <div class="flex gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="setScopeVisible('admin', true)">
                    {{ t('admin.menuManagement.showAll') }}
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" @click="setScopeVisible('admin', false)">
                    {{ t('admin.menuManagement.hideAll') }}
                  </button>
                </div>
              </div>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-dark-700">
              <li
                v-for="item in ADMIN_BUILTIN_MENUS"
                :key="item.path"
                class="flex items-center justify-between gap-4 px-5 py-3"
                :class="{ 'opacity-60': item.path === '/admin/menu' }"
              >
                <div class="min-w-0">
                  <div class="font-medium text-gray-900 dark:text-white">
                    {{ t(item.labelKey) }}
                    <span
                      v-if="item.path === '/admin/menu'"
                      class="ml-2 rounded bg-amber-50 px-1.5 py-0.5 text-[10px] font-normal text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                    >
                      {{ t('admin.menuManagement.thisPage') }}
                    </span>
                  </div>
                  <div class="truncate font-mono text-xs text-gray-400">{{ item.path }}</div>
                </div>
                <div class="flex items-center gap-3">
                  <span
                    class="text-xs"
                    :class="isVisible(item.path) ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400'"
                  >
                    {{ isVisible(item.path) ? t('admin.menuManagement.visible') : t('admin.menuManagement.hidden') }}
                  </span>
                  <Toggle :model-value="isVisible(item.path)" @update:model-value="setVisible(item.path, $event)" />
                </div>
              </li>
            </ul>
          </div>

          <p class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.menuManagement.hiddenCount', { n: hiddenMenuKeys.length }) }}
          </p>
        </div>

        <!-- Custom menus -->
        <div v-show="activeTab === 'custom'" class="space-y-4">
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.settings.customMenu.title') }}
                  </h2>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.customMenu.description') }}
                  </p>
                </div>
                <button type="button" class="btn btn-primary btn-sm" @click="addCustomItem">
                  {{ t('admin.settings.customMenu.add') }}
                </button>
              </div>
            </div>

            <div v-if="customMenuItems.length === 0" class="px-5 py-12 text-center text-sm text-gray-400">
              {{ t('admin.menuManagement.noCustomMenus') }}
            </div>

            <div class="space-y-4 p-5">
              <div
                v-for="(item, index) in customMenuItems"
                :key="item.id || index"
                class="rounded-xl border border-gray-200 p-4 dark:border-dark-600"
              >
                <div class="mb-3 flex items-center justify-between">
                  <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.customMenu.itemLabel', { n: index + 1 }) }}
                  </span>
                  <div class="flex items-center gap-1">
                    <button
                      v-if="index > 0"
                      type="button"
                      class="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                      :title="t('admin.settings.customMenu.moveUp')"
                      @click="moveCustomItem(index, -1)"
                    >
                      ↑
                    </button>
                    <button
                      v-if="index < customMenuItems.length - 1"
                      type="button"
                      class="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                      :title="t('admin.settings.customMenu.moveDown')"
                      @click="moveCustomItem(index, 1)"
                    >
                      ↓
                    </button>
                    <button
                      type="button"
                      class="rounded p-1.5 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                      :title="t('admin.settings.customMenu.remove')"
                      @click="removeCustomItem(index)"
                    >
                      ✕
                    </button>
                  </div>
                </div>

                <div class="grid gap-3 sm:grid-cols-2">
                  <label class="block text-sm">
                    <span class="mb-1 block text-gray-600 dark:text-gray-300">
                      {{ t('admin.settings.customMenu.name') }}
                    </span>
                    <input
                      v-model="item.label"
                      type="text"
                      class="input w-full"
                      :placeholder="t('admin.settings.customMenu.namePlaceholder')"
                      @input="markDirty"
                    />
                  </label>

                  <label class="block text-sm">
                    <span class="mb-1 block text-gray-600 dark:text-gray-300">
                      {{ t('admin.settings.customMenu.visibility') }}
                    </span>
                    <select v-model="item.visibility" class="input w-full" @change="markDirty">
                      <option value="user">{{ t('admin.settings.customMenu.visibilityUser') }}</option>
                      <option value="admin">{{ t('admin.settings.customMenu.visibilityAdmin') }}</option>
                    </select>
                  </label>

                  <label class="block text-sm sm:col-span-2">
                    <span class="mb-1 block text-gray-600 dark:text-gray-300">
                      {{ t('admin.settings.customMenu.url') }}
                    </span>
                    <input
                      v-model="item.url"
                      type="text"
                      class="input w-full font-mono text-sm"
                      :placeholder="t('admin.settings.customMenu.urlPlaceholder')"
                      @input="markDirty"
                    />
                  </label>

                  <label class="block text-sm sm:col-span-2">
                    <span class="mb-1 block text-gray-600 dark:text-gray-300">
                      {{ t('admin.settings.customMenu.iconSvg') }}
                    </span>
                    <textarea
                      v-model="item.icon_svg"
                      rows="2"
                      class="input w-full font-mono text-xs"
                      :placeholder="t('admin.settings.customMenu.iconSvgPlaceholder')"
                      @input="markDirty"
                    />
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AppLayout } from '@/components/layout'
import Toggle from '@/components/common/Toggle.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { adminAPI } from '@/api'
import { useAppStore, useAdminSettingsStore } from '@/stores'
import type { CustomMenuItem } from '@/types'
import {
  ADMIN_BUILTIN_MENUS,
  USER_BUILTIN_MENUS,
  type BuiltinMenuScope,
} from '@/constants/menuCatalog'

const { t } = useI18n()
const appStore = useAppStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(true)
const saving = ref(false)
const dirty = ref(false)
const activeTab = ref<'builtin' | 'custom'>('builtin')

const hiddenMenuKeys = ref<string[]>([])
const customMenuItems = ref<CustomMenuItem[]>([])

const tabs = computed(() => [
  {
    id: 'builtin' as const,
    label: t('admin.menuManagement.tabBuiltin'),
    badge: hiddenMenuKeys.value.length || null,
  },
  {
    id: 'custom' as const,
    label: t('admin.menuManagement.tabCustom'),
    badge: customMenuItems.value.length || null,
  },
])

function markDirty() {
  dirty.value = true
}

function isVisible(path: string): boolean {
  return !hiddenMenuKeys.value.includes(path)
}

function setVisible(path: string, visible: boolean) {
  const set = new Set(hiddenMenuKeys.value)
  if (visible) set.delete(path)
  else set.add(path)
  hiddenMenuKeys.value = Array.from(set)
  markDirty()
}

function setScopeVisible(scope: BuiltinMenuScope, visible: boolean) {
  const list = scope === 'user' ? USER_BUILTIN_MENUS : ADMIN_BUILTIN_MENUS
  const set = new Set(hiddenMenuKeys.value)
  for (const item of list) {
    // Keep menu management page visible when bulk-hiding admin menus.
    if (!visible && item.path === '/admin/menu') continue
    if (visible) set.delete(item.path)
    else set.add(item.path)
  }
  hiddenMenuKeys.value = Array.from(set)
  markDirty()
}

function generateId(): string {
  return `m_${Math.random().toString(36).slice(2, 10)}`
}

function addCustomItem() {
  customMenuItems.value.push({
    id: generateId(),
    label: '',
    icon_svg: '',
    url: '',
    visibility: 'user',
    sort_order: customMenuItems.value.length,
  })
  markDirty()
}

function removeCustomItem(index: number) {
  customMenuItems.value.splice(index, 1)
  customMenuItems.value.forEach((item, i) => {
    item.sort_order = i
  })
  markDirty()
}

function moveCustomItem(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= customMenuItems.value.length) return
  const items = customMenuItems.value
  const tmp = items[index]
  items[index] = items[target]
  items[target] = tmp
  items.forEach((item, i) => {
    item.sort_order = i
  })
  markDirty()
}

async function load() {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getSettings()
    hiddenMenuKeys.value = Array.isArray(settings.hidden_menu_keys)
      ? [...settings.hidden_menu_keys]
      : []
    customMenuItems.value = Array.isArray(settings.custom_menu_items)
      ? settings.custom_menu_items.map((item, i) => ({
          id: item.id || generateId(),
          label: item.label || '',
          icon_svg: item.icon_svg || '',
          url: item.url || '',
          page_slug: item.page_slug,
          visibility: item.visibility === 'admin' ? 'admin' : 'user',
          sort_order: typeof item.sort_order === 'number' ? item.sort_order : i,
        }))
      : []
    dirty.value = false
  } catch (err) {
    console.error('[MenuManagement] load failed', err)
    appStore.showToast('error', t('admin.menuManagement.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (saving.value) return
  // Basic validation for custom items
  for (const item of customMenuItems.value) {
    if (!item.label.trim()) {
      appStore.showToast('error', t('admin.menuManagement.customLabelRequired'))
      activeTab.value = 'custom'
      return
    }
    if (!item.url.trim()) {
      appStore.showToast('error', t('admin.menuManagement.customUrlRequired'))
      activeTab.value = 'custom'
      return
    }
  }

  saving.value = true
  try {
    const updated = await adminAPI.settings.updateSettings({
      hidden_menu_keys: [...hiddenMenuKeys.value],
      custom_menu_items: customMenuItems.value.map((item, i) => ({
        ...item,
        sort_order: i,
      })),
    })

    hiddenMenuKeys.value = Array.isArray(updated.hidden_menu_keys)
      ? [...updated.hidden_menu_keys]
      : [...hiddenMenuKeys.value]
    customMenuItems.value = Array.isArray(updated.custom_menu_items)
      ? updated.custom_menu_items
      : customMenuItems.value

    // Sync stores so sidebar updates immediately
    adminSettingsStore.customMenuItems = customMenuItems.value
    adminSettingsStore.hiddenMenuKeys = hiddenMenuKeys.value
    if (appStore.cachedPublicSettings) {
      appStore.cachedPublicSettings = {
        ...appStore.cachedPublicSettings,
        hidden_menu_keys: [...hiddenMenuKeys.value],
        custom_menu_items: customMenuItems.value.filter((i) => i.visibility === 'user'),
      }
    }

    dirty.value = false
    appStore.showToast('success', t('admin.menuManagement.saveSuccess'))
  } catch (err) {
    console.error('[MenuManagement] save failed', err)
    appStore.showToast('error', t('admin.menuManagement.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
