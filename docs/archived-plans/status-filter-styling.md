# Plan: Style Status Filter Buttons to Match Theme

## Context

The status filter buttons on `CustomersPage.svelte` (lines 282-290) currently use a minimal style that doesn't match the project's dark theme with violet-indigo accent palette. The buttons need to visually integrate with the established design system used across other pages (ProductsPage, TransactionsPage).

## Current State

Lines 282-290 in `CustomersPage.svelte`:
```svelte
<div class="flex gap-2">
  <button class="btn btn-sm py-1 px-3 rounded hover:bg-surface-hover {statusFilter === 'all' ? 'bg-primary-subtle/10' : ''}" onclick={...}>
    All
  </button>
  <button class="btn btn-sm py-1 px-3 rounded hover:bg-surface-hover {statusFilter === 'active' ? 'bg-primary-subtle/10' : ''}" onclick={...}>
    Active
  </button>
  <button class="btn btn-sm py-1 px-3 rounded hover:bg-surface-hover {statusFilter === 'inactive' ? 'bg-primary-subtle/10' : ''}" onclick={...}>
    Inactive
  </button>
</div>
```

Issues:
- Uses `bg-primary-subtle/10` (very subtle, barely visible) for active state
- No border styling — flat appearance doesn't match the card-based layout
- Inconsistent with the filter button patterns seen in ProductsPage (which uses explicit border + background styling with `rgba(124,58,236,0.12)` and `rgba(124,58,236,0.35)`)

## Design System Reference

From `app.css`:
- `--color-primary-subtle: rgba(124, 58, 237, 0.15)`
- `--color-primary-default: #7c3aed`
- `--color-primary-light: #8b5cf6`
- `--color-surface-default: #111827`
- `--color-surface-hover: #1a2540`
- `--color-border-default: #1e293b`
- `--color-border-strong: #334155`
- `--color-text-muted: #64748b`
- `--color-text-primary: #e2e8f0`
- `--color-success: #10b981`
- `--color-success-subtle: rgba(16, 185, 129, 0.12)`
- `--color-success-light: #34d399`
- `--color-danger: #f43f5e`
- `--color-danger-subtle: rgba(244, 63, 94, 0.12)`
- `--color-danger-light: #fb7185`

From ProductsPage filter button pattern (line 544):
```svelte
class="... border {lowStockOnly ? 'bg-warning/10 border-warning/30 text-warning-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
```

## Proposed Change

Replace the three status filter buttons with a styled segmented control that matches the dark theme:

**Active state per button:**
- `All` active: `bg-primary-subtle border-primary-default/30 text-primary-light`
- `Active` active: `bg-success-subtle border-success-default/30 text-success-light`
- `Inactive` active: `bg-danger-subtle border-danger-default/30 text-danger-light`

**Inactive state (all buttons):**
- `bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong hover:bg-surface-hover`

**Container:**
- Wrap in a flex container with `p-1 bg-bg-secondary rounded-xl border border-border-default` to create a segmented control appearance

**Button styling:**
- Use `h-8 px-3.5 rounded-lg text-xs font-medium transition-all duration-200` for consistent sizing
- Remove `btn btn-sm` classes to avoid conflicting with the custom segmented style

## Files to Modify

- `web/src/lib/pages/CustomersPage.svelte` — lines 281-291 (the filter buttons section)

## Implementation

Replace lines 281-291 with:

```svelte
<div class="flex items-center p-1 bg-bg-secondary rounded-xl border border-border-default">
  <button
    class="h-8 px-3.5 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light shadow-glow-primary-sm' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
    onclick={() => { statusFilter = 'all'; handleStatusFilterChange(); }}
  >
    All
  </button>
  <button
    class="h-8 px-3.5 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
    onclick={() => { statusFilter = 'active'; handleStatusFilterChange(); }}
  >
    Active
  </button>
  <button
    class="h-8 px-3.5 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
    onclick={() => { statusFilter = 'inactive'; handleStatusFilterChange(); }}
  >
    Inactive
  </button>
</div>
```

This approach:
- Uses a segmented control container (`bg-bg-secondary rounded-xl border border-border-default`) matching the dark theme
- Active "All" gets the primary violet glow (consistent with the app's accent color)
- Active "Active" gets success green (semantic match)
- Active "Inactive" gets danger red (semantic match)
- Inactive buttons use muted text with subtle hover states
- All transitions use `duration-200` for smooth feel
