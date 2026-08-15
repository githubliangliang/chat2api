import { USER_BUILTIN_MENUS } from '@/constants/menuCatalog'

export const ADMIN_HOME_PATH = '/admin/accounts'
export const USER_HOME_FALLBACK = '/keys'

const SIMPLE_MODE_USER_SKIPS = ['/batch-image', '/usage', '/available-channels'] as const

export function collectHiddenMenuKeys(
  ...sources: Array<readonly string[] | null | undefined>
): string[] {
  const hidden = new Set<string>()
  for (const src of sources) {
    for (const raw of src ?? []) {
      const key = String(raw || '').trim()
      if (key) hidden.add(key)
    }
  }
  return [...hidden]
}

export function resolveUserHomePath(opts?: {
  hiddenMenuKeys?: Iterable<string>
  skipPaths?: Iterable<string>
  isSimpleMode?: boolean
}): string {
  const hidden = new Set<string>()
  const add = (value: string) => {
    const key = String(value || '').trim()
    if (key) hidden.add(key)
  }
  for (const key of opts?.hiddenMenuKeys ?? []) add(key)
  for (const key of opts?.skipPaths ?? []) add(key)
  if (opts?.isSimpleMode) {
    for (const path of SIMPLE_MODE_USER_SKIPS) hidden.add(path)
  }
  for (const item of USER_BUILTIN_MENUS) {
    if (!hidden.has(item.path)) return item.path
  }
  return USER_HOME_FALLBACK
}

export function resolveSignedInHomePath(opts: {
  isAdmin: boolean
  hiddenMenuKeys?: Iterable<string>
  skipPaths?: Iterable<string>
  isSimpleMode?: boolean
}): string {
  if (opts.isAdmin) return ADMIN_HOME_PATH
  return resolveUserHomePath(opts)
}

export function sanitizeInternalRedirectPath(
  path: string | null | undefined,
  fallback: string = resolveUserHomePath(),
): string {
  if (!path) return fallback
  if (!path.startsWith('/')) return fallback
  if (path.startsWith('//')) return fallback
  if (path.includes('://')) return fallback
  if (path.includes('\n') || path.includes('\r')) return fallback
  return path
}
