import { describe, expect, it } from 'vitest'
import { ADMIN_BUILTIN_MENUS } from '@/constants/menuCatalog'

describe('ADMIN_BUILTIN_MENUS', () => {
  it('includes model plaza so Menu Management can hide it', () => {
    expect(ADMIN_BUILTIN_MENUS).toContainEqual({
      path: '/admin/model-plaza',
      labelKey: 'nav.modelPlaza',
      scope: 'admin'
    })
  })
})
