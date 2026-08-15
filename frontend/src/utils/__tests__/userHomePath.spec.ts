import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import {
  ADMIN_HOME_PATH,
  USER_HOME_FALLBACK,
  collectHiddenMenuKeys,
  resolveSignedInHomePath,
  resolveUserHomePath,
  sanitizeInternalRedirectPath,
} from '@/utils/userHomePath'
import { currentSignedInHomePath, resolveDefaultUserHomePath } from '@/utils/userHomePathStore'
import { resolveCompletedSetupRedirectPath } from '@/router/setupRedirect'

describe('resolveUserHomePath', () => {
  it('defaults to /dashboard', () => {
    expect(resolveUserHomePath()).toBe('/dashboard')
  })

  it('skips a hidden dashboard and lands on /keys', () => {
    expect(resolveUserHomePath({ hiddenMenuKeys: ['/dashboard'] })).toBe('/keys')
  })

  it('trims hidden keys', () => {
    expect(resolveUserHomePath({ hiddenMenuKeys: [' /dashboard '] })).toBe('/keys')
  })

  it('walks the builtin catalog until a visible item remains', () => {
    expect(
      resolveUserHomePath({
        hiddenMenuKeys: ['/dashboard', '/keys', '/batch-image'],
      }),
    ).toBe('/usage')
  })

  it('skips simple-mode user pages when choosing a fallback', () => {
    expect(
      resolveUserHomePath({
        hiddenMenuKeys: ['/dashboard', '/keys'],
        isSimpleMode: true,
      }),
    ).toBe('/profile')
  })

  it('returns /keys when every builtin user menu is hidden', () => {
    expect(
      resolveUserHomePath({
        hiddenMenuKeys: [
          '/dashboard',
          '/keys',
          '/batch-image',
          '/usage',
          '/available-channels',
          '/profile',
        ],
      }),
    ).toBe(USER_HOME_FALLBACK)
  })
})

describe('resolveSignedInHomePath', () => {
  it('sends admins to the account list regardless of hidden user menus', () => {
    expect(
      resolveSignedInHomePath({
        isAdmin: true,
        hiddenMenuKeys: ['/dashboard'],
      }),
    ).toBe(ADMIN_HOME_PATH)
  })

  it('uses the user catalog for non-admins', () => {
    expect(
      resolveSignedInHomePath({
        isAdmin: false,
        hiddenMenuKeys: ['/dashboard'],
      }),
    ).toBe('/keys')
  })
})

describe('sanitizeInternalRedirectPath', () => {
  it('keeps an explicit /dashboard even when it is the hidden default', () => {
    expect(
      sanitizeInternalRedirectPath(
        '/dashboard',
        resolveUserHomePath({ hiddenMenuKeys: ['/dashboard'] }),
      ),
    ).toBe('/dashboard')
  })

  it('falls back when the path is empty or unsafe', () => {
    const fallback = '/keys'
    expect(sanitizeInternalRedirectPath('', fallback)).toBe(fallback)
    expect(sanitizeInternalRedirectPath('https://evil.test', fallback)).toBe(fallback)
    expect(sanitizeInternalRedirectPath('//evil.test', fallback)).toBe(fallback)
    expect(sanitizeInternalRedirectPath('/ok\n/x', fallback)).toBe(fallback)
  })
})

describe('collectHiddenMenuKeys', () => {
  it('merges public and admin sources', () => {
    expect(collectHiddenMenuKeys(['/dashboard'], ['/usage', '/dashboard'])).toEqual([
      '/dashboard',
      '/usage',
    ])
  })
})

describe('store-aware home helpers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('reads hidden_menu_keys from public settings', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      hidden_menu_keys: ['/dashboard'],
    } as typeof appStore.cachedPublicSettings
    expect(resolveDefaultUserHomePath()).toBe('/keys')
    expect(currentSignedInHomePath()).toBe('/keys')
  })

  it('sends signed-in admins to the admin home', () => {
    const authStore = useAuthStore()
    const adminSettingsStore = useAdminSettingsStore()
    authStore.user = { role: 'admin' } as typeof authStore.user
    adminSettingsStore.hiddenMenuKeys = ['/dashboard']
    expect(currentSignedInHomePath()).toBe(ADMIN_HOME_PATH)
  })
})

describe('resolveCompletedSetupRedirectPath', () => {
  it('follows the signed-in home helper', () => {
    expect(resolveCompletedSetupRedirectPath(true, true, ['/dashboard'])).toBe(ADMIN_HOME_PATH)
    expect(resolveCompletedSetupRedirectPath(true, false, ['/dashboard'])).toBe('/keys')
    expect(resolveCompletedSetupRedirectPath(false, false)).toBe('/dashboard')
  })
})
