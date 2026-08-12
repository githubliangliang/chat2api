/**
 * Built-in sidebar menu catalog used by Menu Management UI.
 * path must match AppSidebar.vue NavItem.path values.
 */

export type BuiltinMenuScope = 'user' | 'admin'

export interface BuiltinMenuEntry {
  /** Stable path used as hidden_menu_keys value */
  path: string
  /** i18n key under nav.* */
  labelKey: string
  scope: BuiltinMenuScope
  /** Optional note for UI */
  noteKey?: string
}

/** User-facing built-in menus (regular user sidebar + admin "My Account") */
export const USER_BUILTIN_MENUS: BuiltinMenuEntry[] = [
  { path: '/dashboard', labelKey: 'nav.dashboard', scope: 'user' },
  { path: '/keys', labelKey: 'nav.apiKeys', scope: 'user' },
  { path: '/batch-image', labelKey: 'nav.batchImage', scope: 'user' },
  { path: '/usage', labelKey: 'nav.usage', scope: 'user' },
  { path: '/available-channels', labelKey: 'nav.availableChannels', scope: 'user' },
  { path: '/profile', labelKey: 'nav.profile', scope: 'user' },
]

/** Admin-facing built-in menus */
export const ADMIN_BUILTIN_MENUS: BuiltinMenuEntry[] = [
  { path: '/admin/users', labelKey: 'nav.users', scope: 'admin' },
  { path: '/admin/groups', labelKey: 'nav.groups', scope: 'admin' },
  { path: '/admin/channels', labelKey: 'nav.channelManagement', scope: 'admin' },
  { path: '/admin/channels/pricing', labelKey: 'nav.channelPricing', scope: 'admin' },
  { path: '/admin/channels/monitor', labelKey: 'nav.channelMonitor', scope: 'admin' },
  { path: '/admin/subscriptions', labelKey: 'nav.subscriptions', scope: 'admin' },
  { path: '/admin/accounts', labelKey: 'nav.accounts', scope: 'admin' },
  { path: '/admin/proxies', labelKey: 'nav.proxies', scope: 'admin' },
  { path: '/admin/risk-control', labelKey: 'nav.contentModeration', scope: 'admin' },
  { path: '/admin/prompt-audit', labelKey: 'nav.promptAudit', scope: 'admin' },
  { path: '/admin/usage', labelKey: 'nav.usage', scope: 'admin' },
  { path: '/admin/audit-logs', labelKey: 'nav.auditLogs', scope: 'admin' },
  { path: '/admin/settings', labelKey: 'nav.settings', scope: 'admin' },
  { path: '/admin/menu', labelKey: 'nav.menuManagement', scope: 'admin' },
]

export const ALL_BUILTIN_MENUS: BuiltinMenuEntry[] = [
  ...USER_BUILTIN_MENUS,
  ...ADMIN_BUILTIN_MENUS,
]
