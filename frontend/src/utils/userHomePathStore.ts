import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import {
  collectHiddenMenuKeys,
  resolveSignedInHomePath,
  resolveUserHomePath,
} from '@/utils/userHomePath'

/** Public-settings-aware user home. Safe before login (ignores admin role). */
export function resolveDefaultUserHomePath(): string {
  const appStore = useAppStore()
  return resolveUserHomePath({
    hiddenMenuKeys: appStore.cachedPublicSettings?.hidden_menu_keys,
  })
}

/** Logo / login / guard fallback for the current session. */
export function currentSignedInHomePath(): string {
  const authStore = useAuthStore()
  const appStore = useAppStore()
  const adminSettingsStore = useAdminSettingsStore()
  return resolveSignedInHomePath({
    isAdmin: authStore.isAdmin,
    isSimpleMode: authStore.isSimpleMode,
    hiddenMenuKeys: collectHiddenMenuKeys(
      appStore.cachedPublicSettings?.hidden_menu_keys,
      adminSettingsStore.hiddenMenuKeys,
    ),
  })
}
