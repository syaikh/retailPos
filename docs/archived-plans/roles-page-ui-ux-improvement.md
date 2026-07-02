# Roles Page UI/UX Improvement Plan

## 1. Analysis of Current State

The Roles Page (`web/src/lib/pages/admin/RolesPage.svelte`) uses a **card-based layout** where each role is a large card showing role metadata + a grid of permission badges. The modal-based create/edit flow uses collapsible permission groups.

### Existing Design System
- **Dark theme**: Deep space palette (`#080c14` bg, `#111827` surface, violet-indigo accent `#7c3aed`)
- **Components**: `Modal` (sm/md/lg/xl), `Badge` (variant/size), `Skeleton`, `Pagination`
- **Patterns**: Tables with `table-fixed`, cards with `bg-surface/40 backdrop-blur-xl`, action buttons with `btn-icon`
- **Spacing**: `space-y-5` page-level, `gap-6` between cards, `p-6` card padding
- **Typography**: `text-3xl font-extrabold` headings, `text-lg font-bold` card titles, `text-sm` body

---

## 2. Identified Problems & Impact

### 2.1 Information Hierarchy
| Problem | Impact |
|---|---|
| Role cards use `text-3xl font-extrabold` for page heading but `text-lg font-bold` for role names — hierarchy is flat | Users can't quickly scan role names; heading competes with content |
| Permission badges show raw `permCode` (e.g., `user:read`) alongside Indonesian name — duplicate information at same visual weight | Cognitive overload; codes should be secondary |
| "System" badge and "N Perms" badge are same size/weight as role name | Metadata competes with primary content |
| No visual separator between role info and permission grid | Hard to distinguish metadata from permissions |

### 2.2 Table Usability (List View)
| Problem | Impact |
|---|---|
| Card-based layout wastes horizontal space on desktop | Only ~40% of width used for actual content; excessive scrolling |
| No sort, no pagination for roles list | With 10+ roles, finding specific role requires scanning entire page |
| Permission grid uses `grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4` — uneven wrapping | Permissions reflow unpredictably; hard to scan |
| Hover states only on card level (`hover:border-primary/30`), not on individual actions | Users can't discover interactive elements |
| No description truncation or expansion | Long descriptions push permission grid down |

### 2.3 Search & Filtering
| Problem | Impact |
|---|---|
| No search on the roles list page itself | Finding a role requires manual scanning |
| No filter by permission type or system status | Can't narrow down roles |
| Modal search only filters permissions, not roles | Two different search contexts with different behaviors |

### 2.4 Permission Management (Modal)
| Problem | Impact |
|---|---|
| Permission groups collapsed by default — but no count of total permissions visible until expanded | Users don't know what they're missing |
| `max-h-64` scroll container for permissions is too short | Constant scrolling even with groups collapsed |
| Permission checkboxes have no visual grouping within groups | Flat list of checkboxes doesn't show relationships |
| "Select All" per group is hidden when collapsed | Users must expand to discover bulk actions |
| No summary of selected permissions before save | Users can't review before committing |

### 2.5 Role Creation/Editing Workflow
| Problem | Impact |
|---|---|
| Modal treats create and edit identically except for name field | Edit mode should focus on permissions, not re-show name |
| No keyboard shortcut to save (Ctrl+Enter) | Forces mouse usage |
| No draft/auto-save | Accidental close loses all work |
| Permission count shows "0 of 31" — total doesn't update during search | Confusing when searching shows filtered count but total stays same |

### 2.6 Empty/Loading/Error States
| Problem | Impact |
|---|---|
| Loading skeleton shows 3 placeholder cards — doesn't match actual card layout | Jarring transition when real content loads |
| Empty state has no illustration, just icon + text | Doesn't guide user to next action |
| No error state for API failures beyond toast | Toast disappears; user doesn't know what failed |
| No pagination loading state | Users don't know if more roles exist |

### 2.7 Action Buttons
| Problem | Impact |
|---|---|
| Edit and Delete buttons are in a floating `bg-surface-subtle/30` container — easy to miss | Low discoverability |
| Delete button has no confirmation count ("This will affect N users") | Dangerous action without context |
| No bulk actions (delete multiple roles) | Enterprise inefficiency |
| "Create Role" button is alone in header — no refresh or view options | Limited control |

### 2.8 Accessibility
| Problem | Impact |
|---|---|
| No visible focus indicators on group toggle buttons | Keyboard users can't see where they are |
| `aria-live` region for search results uses `sr-only` class but is inside sticky header | May not be announced properly |
| Permission checkboxes lack `aria-describedby` linking to description | Screen readers don't get context |
| No `role="list"` or `role="listitem"` on roles collection | Not announced as a list |
| Color-only indicators (System badge color) | Color-blind users can't distinguish |

---

## 3. Proposed Improvements

### 3.1 Layout: Card → Table Hybrid

**Convert the roles list to a table layout** for desktop, keeping cards for mobile:

```
Desktop (≥1024px):
┌─────────────────────────────────────────────────────────────────┐
│  Roles Management                    [+ Create Role]  [Refresh] │
├─────────────────────────────────────────────────────────────────┤
│  Search roles... [Filter ▾] [System ▾] [Sort ▾]                │
├──────────────┬──────────┬────────────┬────────────┬────────────┤
│ ROLE         │ SYSTEM   │ PERMISSIONS│ USERS      │ ACTIONS    │
├──────────────┼──────────┼────────────┼────────────┼────────────┤
│ admin        │ ● System │ 30 perms   │ 3 users    │ ✏️  🗑️      │
│ manager      │          │ 12 perms   │ 5 users    │ ✏️  🗑️      │
│ cashier      │ ● System │ 6 perms    │ 8 users    │ ✏️          │
└──────────────┴──────────┴────────────┴────────────┴────────────┘

Mobile (<768px): Card layout (collapsed)
┌─────────────────────────────────────┐
│  admin                    ✏️  🗑️     │
│  ● System  │  30 Perms  │  3 Users │
└─────────────────────────────────────┘
```

**Why**: Tables are the enterprise standard for data-dense lists. Users expect sortable columns, consistent row heights, and scanable layouts. Cards waste space on desktop.

### 3.2 Information Hierarchy

| Element | Before | After |
|---|---|---|
| Page heading | `text-3xl font-extrabold` | `text-2xl font-bold` — less dominant |
| Role name | `text-lg font-bold` | `text-base font-semibold text-text-primary` — clean, scannable |
| Role description | `text-sm text-text-muted` | `text-sm text-text-muted truncate max-w-xs` — prevent overflow |
| Permission count | Same weight as name | `Badge variant="muted" size="sm"` — secondary |
| System badge | `text-[10px] uppercase tracking-widest` | `Badge variant="primary" size="sm"` — consistent with design system |
| Permission codes | `text-[10px] uppercase tracking-wider` | `text-[10px] text-text-muted font-mono` — clearly secondary |

### 3.3 Search & Filtering

**Add to roles list page:**
- Search input with debounce (match UsersPage pattern)
- Filter dropdown: All Roles / System Roles / Custom Roles
- Sort dropdown: Name A-Z / Name Z-A / Most Permissions / Fewest Permissions
- Active filter pills with clear buttons (match UsersPage pattern)

**Why**: The UsersPage already has this pattern. Consistency reduces learning curve.

### 3.4 Permission Management (Modal)

| Change | Rationale |
|---|---|
| Increase scroll container to `max-h-80` | Less scrolling, more visible permissions |
| Show permission count in collapsed header: "Inventory (3)" | Users know what's inside without expanding |
| Move "Select All" to always-visible header (not inside expand) | Bulk actions always accessible |
| Add permission description tooltip on hover/focus | `title` attribute already exists; add `aria-describedby` |
| Add summary panel at bottom: "7 of 31 permissions selected across 3 groups" | Clear save-time confirmation |
| Group permissions by action type (CRUD) within each category | `user:read, user:view` → "Read" group; `user:create, user:update, user:delete` → "Write" group |
| Add inline indeterminate state for "Select All" when some selected | Visual feedback for partial selection |

### 3.5 Role Creation/Editing Workflow

| Change | Rationale |
|---|---|
| Split create into 2-step: (1) Name + Description → (2) Permissions | Reduces cognitive load; name is quick, permissions need focus |
| Edit mode: Show role name as non-editable header, go straight to permissions | Faster edit path |
| Add Ctrl+Enter to save | Keyboard power-user support |
| Show unsaved changes indicator in modal header (dot icon) | Constant awareness of dirty state |
| Add "Duplicate Role" action | Common enterprise workflow |

### 3.6 Empty/Loading/Error States

| State | Before | After |
|---|---|---|
| Loading | 3 skeleton cards | Table skeleton with 5 rows matching actual columns |
| Empty (no roles) | Icon + text + button | Icon + heading + description + primary CTA + secondary "Learn more" link |
| Empty (no search results) | N/A | "No roles match 'xxx'" + clear search button |
| Error | Toast only | Inline error banner with retry button in content area |
| Delete error | Toast only | Inline error with explanation + dismiss |

### 3.7 Action Buttons

| Change | Rationale |
|---|---|
| Edit: Always visible as `btn-icon` with `aria-label` | Don't hide primary action |
| Delete: `btn-icon` with danger color on hover, confirmation shows affected user count | Safety + context |
| Add "Refresh" button next to Create | Explicit refresh control |
| Action column pinned right on desktop, dropdown on mobile | Consistent with ProductActionsDropdown pattern |

### 3.8 Accessibility

| Change | Rationale |
|---|---|---|
| Add `role="list"` to table body, `role="listitem"` to rows | Screen reader context |
| Visible focus ring on all interactive elements (`focus-visible:ring-2`) | Keyboard navigation |
| `aria-describedby` on permission checkboxes linking to description | Context for screen readers |
| System status: Add icon (🔒) alongside color | Non-color indicator |
| Permission groups: `aria-expanded` + `aria-controls` + `role="group"` | Proper ARIA pattern |
| Add `aria-busy="true"` on save button during submission | Loading state announcement |

### 3.9 Responsive Design

| Breakpoint | Layout |
|---|---|
| ≥1200px | Full table with all columns |
| ≥768px | Table with collapsed actions column |
| <768px | Card layout with expand/collapse |
| <480px | Single column cards, stacked actions |

---

## 4. Implementation Plan

### Phase 1: Foundation (List View)
1. Convert card layout to responsive table/card hybrid
2. Add search input with debounce
3. Add filter dropdowns (system/custom, sort)
4. Add filter pills with clear
5. Improve information hierarchy (badge weights, typography)
6. Update loading skeleton to match table layout
7. Add empty/error states

### Phase 2: Permission Modal
8. Redesign permission groups (always-visible controls, CRUD sub-grouping)
9. Increase scroll container height
10. Add permission summary panel
11. Add Ctrl+Enter save shortcut
12. Add unsaved changes indicator in header
13. Split create into 2-step wizard

### Phase 3: Actions & Workflow
14. Add "Duplicate Role" action
15. Improve delete confirmation with user count
16. Add refresh button
17. Inline error banners for API failures

### Phase 4: Accessibility & Polish
18. Add ARIA roles and labels
19. Add visible focus indicators
20. Add non-color status indicators
21. Final contrast and spacing audit

---

## 5. Design Tokens to Use

All from existing `--theme` in `app.css`:
- `--color-surface-default`, `--color-surface-hover`, `--color-surface-subtle`
- `--color-border-default`, `--color-border-subtle`, `--color-border-strong`
- `--color-primary-default`, `--color-primary-light`, `--color-primary-subtle`
- `--color-text-primary`, `--color-text-secondary`, `--color-text-muted`
- `--color-danger-default`, `--color-danger-subtle`
- `--shadow-glow-primary-sm`, `--shadow-card`
- `--radius-xl` (0.875rem), `--radius-2xl` (1.125rem)

Component patterns from existing code:
- `btn btn-primary`, `btn btn-secondary`, `btn-icon` (from UsersPage)
- `Badge` variant/size (from ProductsPage)
- `Skeleton` widths (from UsersPage)
- `Modal` sizes (lg for permissions, sm for confirmations)
- `card` class with `overflow-hidden` for table containment

---

## 6. Risk Assessment

| Risk | Mitigation |
|---|---|
| Table layout breaks existing tests | Update E2E tests after implementation; table structure is simpler to test |
| Two-step create modal changes UX significantly | Make it optional — auto-advance after name is valid |
| Permission count query adds API load | Use existing `roles` data; count from permissions array |
| Responsive table adds complexity | Use CSS `display: none` on columns, not separate markup |
