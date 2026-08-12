export function resolveCompletedSetupRedirectPath(
  _isAuthenticated: boolean,
  isAdmin: boolean,
): string {
  // Admin home after removing /admin/dashboard
  return isAdmin ? '/admin/accounts' : '/dashboard'
}
