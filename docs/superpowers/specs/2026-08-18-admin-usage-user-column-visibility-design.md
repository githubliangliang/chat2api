# Admin Usage User Column Visibility

## Goal

Allow administrators to hide the user column in both tables on `/admin/usage`:

- Usage details
- Error requests

The user column remains visible by default.

## Design

Both tables already derive their column-settings menu from a list of columns that excludes columns marked as always visible. Remove `user` from each table's always-visible list while leaving it in the full column list.

No new state or UI is required. The existing toggle handlers will hide or show the user column and persist the choice. Usage details and error requests continue to use their existing independent `localStorage` keys, so changing one table does not affect the other.

Existing saved preferences require no migration. A saved hidden-column list that does not contain `user` continues to show the column; after the user hides it, the existing persistence mechanism adds `user` to that table's saved list.

## Verification

Add focused `UsageView` component tests that verify:

- The usage-details column menu includes the user column and can hide it.
- The error-requests column menu includes the user column and can hide it.
- Each action persists `user` through the corresponding existing storage key.
- The user column remains visible by default when no preference is saved.

Run the focused component test file, then the frontend type check or build used by the repository.
