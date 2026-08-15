import { resolveSignedInHomePath } from '@/utils/userHomePath'

export function resolveCompletedSetupRedirectPath(
  _isAuthenticated: boolean,
  isAdmin: boolean,
  hiddenMenuKeys?: Iterable<string>,
  isSimpleMode?: boolean,
): string {
  return resolveSignedInHomePath({ isAdmin, hiddenMenuKeys, isSimpleMode })
}
