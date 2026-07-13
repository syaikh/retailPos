# UI/UX Audit — Retail POS System (First Audit)

**Date:** 2026-07-12
**Auditor:** opencode (big-pickle)
**Previous Score:** N/A (first audit)
**Current Score:** 6.8/10
**Accessibility Score:** 5.5/10

---

## Executive Summary

The RetailPOS frontend is a well-built, modern dark-theme POS system with strong design system foundations, consistent component reuse, and thoughtful POS-specific keyboard shortcuts. The codebase demonstrates solid Svelte 5 patterns and a coherent visual identity.

However, there are meaningful gaps in accessibility, mobile UX, input consistency, and internationalization. The most critical issues center on the CheckoutModal's broken keyboard trap, missing focus management in several custom modals, and the absence of ARIA attributes on interactive table elements.

---

## Score Breakdown

| Category | Score | Notes |
|---|---|---|
| Accessibility | 5.5 | Shared Modal is great; custom modals break it |
| Responsive Design | 7.0 | Desktop/tablet good; mobile cart UX needs work |
| Consistency | 7.0 | Strong design system; language mixing is the main issue |
| Error Handling UX | 6.5 | Toasts everywhere; no inline validation |
| Navigation | 6.5 | Custom router is functional but limited |
| Forms | 6.0 | Good structure; missing inline validation |
| Data Display | 7.0 | Tables are well-structured; sortable headers are good |
| Performance UX | 6.5 | Skeletons used but missing during page transitions |
| Internationalization | 4.0 | No i18n framework; hardcoded strings |
| Modals & Dialogs | 6.0 | Shared Modal is good; custom modals lack features |
| Toast Notifications | 7.5 | Clean design; minor positioning issues |
| Dark/Light Mode | 5.0 | Dark-only; no theme switching |
| Print/Receipt | 7.0 | Functional; receipt is clean |
| Color & Contrast | 6.0 | Muted text fails WCAG AA; some inline colors |
| **Overall** | **6.8** | |

---

## All Findings

### Accessibility (10 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| A-01 | **Critical** | CheckoutModal missing focus trap | `CheckoutModal.svelte:51-201` |
| A-02 | **High** | CheckoutModal no keyboard close (Escape) | `CheckoutModal.svelte:51` |
| A-03 | **High** | CustomerSelectModal missing focus trap | `CustomerSelectModal.svelte:22` |
| A-04 | **High** | CustomerSelectModal no Escape key support | `CustomerSelectModal.svelte:22` |
| A-05 | **High** | Cart quantity input missing ARIA label | `CartPanel.svelte:98-124` |
| A-06 | **Medium** | Payment method buttons lack `aria-pressed` | `CartPanel.svelte:165-173` |
| A-07 | **Medium** | Quick cash preset buttons lack ARIA labels | `CheckoutModal.svelte:131-145` |
| A-08 | **Medium** | Product table rows not keyboard accessible | `PosProductTable.svelte:63-130` |
| A-09 | **Medium** | Table rows missing `aria-label` | `TransactionTable.svelte:99-104` |
| A-10 | **Low** | Loading spinner in toast lacks `aria-hidden` | `ConfirmDeleteModal.svelte:57` |

### Responsive Design (5 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| R-01 | **High** | Cart panel hidden on mobile with no floating CTA | `PosPage.svelte:489-527` |
| R-02 | **Medium** | CheckoutModal not optimized for mobile | `CheckoutModal.svelte:53-67` |
| R-03 | **Medium** | Date picker dropdown can overflow viewport | `TransactionFilters.svelte:344-394` |
| R-04 | **Medium** | Report page KPI cards lack mobile stacking | `ReportsPage.svelte:460` |
| R-05 | **Low** | CartPanel hardcoded max-h clips on short screens | `CartPanel.svelte:48` |

### Consistency (5 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| C-01 | **High** | Mixed language: Indonesian and English in same UI | Multiple files |
| C-02 | **Medium** | CheckoutModal bypasses shared Modal | `CheckoutModal.svelte` |
| C-03 | **Medium** | CustomerSelectModal bypasses shared Modal | `CustomerSelectModal.svelte` |
| C-04 | **Medium** | Cash input inconsistent styling across modals | `CheckoutModal.svelte:119-127` |
| C-05 | **Low** | Mixed Svelte 4 vs 5 event syntax | `PosPage.svelte:465` |

### Error Handling UX (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| E-01 | **Medium** | Validation errors use only toast, not inline | `ProductsPage.svelte:399-413` |
| E-02 | **Medium** | Cart max-stock toast easily missed | `CartPanel.svelte:155` |
| E-03 | **Low** | Dashboard silently swallows fetch errors | `Home.svelte:36-37` |

### Navigation (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| N-01 | **Medium** | No deep-linking support (custom router) | `router/index.ts` |
| N-02 | **Medium** | Breadcrumb shows raw path for unknown routes | `Topbar.svelte:70` |
| N-03 | **Low** | Sidebar collapse state not persisted | `Sidebar.svelte:17` |

### Forms (4 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| F-01 | **Medium** | Checkout cash input has no focus ring | `CheckoutModal.svelte:119-127` |
| F-02 | **Medium** | Product form price input missing `min` attr | `ProductFormModal.svelte:166-167` |
| F-03 | **Low** | `text-destructive` class not defined in theme | `ProductFormModal.svelte:69,73,84,165,173` |
| F-04 | **Low** | Cart quantity allows keyboard bypass of max stock | `CartPanel.svelte:98-124` |

### Data Display (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| D-01 | **Medium** | POS product table uses `table-fixed` causing truncation | `PosProductTable.svelte:53` |
| D-02 | **Medium** | Transaction drawer status badge has inline CSS | `TransactionDrawer.svelte:90` |
| D-03 | **Low** | Revenue table uses raw emoji for sort indicators | `SortableHeader.svelte:38` |

### Performance UX (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| P-01 | **Medium** | No loading skeleton during page transitions | `Layout.svelte:47-56` |
| P-02 | **Low** | Dashboard stat cards animate on every render | `Home.svelte:112-156` |
| P-03 | **Low** | `scroll-behavior: smooth` despite `prefers-reduced-motion` | `app.css:296` |

### Internationalization (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| I-01 | **High** | No i18n framework; strings are hardcoded | All files |
| I-02 | **Medium** | Currency formatting assumes `id-ID` everywhere | Multiple files |
| I-03 | **Medium** | No RTL support consideration | `app.css`, layout files |

### Modals & Dialogs (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| M-01 | **High** | CheckoutModal z-index management conflicts | `CheckoutModal.svelte:53,62` |
| M-02 | **Medium** | CustomerSelectModal z-index higher than CheckoutModal | `CustomerSelectModal.svelte:22,24` |
| M-03 | **Medium** | Nested modal z-index escalation pattern | Multiple files |

### Toast Notifications (2 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| T-01 | **Medium** | Toast position not responsive on mobile | `Toast.svelte:21` |
| T-02 | **Low** | Error toast same auto-dismiss as success (4s) | `toast.svelte.ts:15` |

### Dark/Light Mode (2 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| DL-01 | **Medium** | No light mode — dark theme only | `app.css:129-202` |
| DL-02 | **Low** | Print styles assume white background | `app.css:21-123` |

### Print/Receipt (2 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| PR-01 | **Medium** | Receipt print uses 300ms delay hack | `PosPage.svelte:272-275` |
| PR-02 | **Low** | Receipt footer text is Indonesian only | `ReceiptPrintOverlay.svelte:61-62` |

### Color & Contrast (3 findings)

| ID | Severity | Title | Location |
|---|---|---|---|
| CC-01 | **Medium** | `text-muted` on dark bg fails WCAG AA (4.3:1) | `app.css:187` |
| CC-02 | **Medium** | CheckoutModal uses `text-purple-400` outside theme | `CheckoutModal.svelte:88` |
| CC-03 | **Low** | Scrollbar thumb very low contrast | `app.css:319` |

---

## Top 5 UX Improvements by Impact

### 1. Migrate CheckoutModal & CustomerSelectModal to Shared Modal
Addresses A-01, A-02, A-03, A-04, C-02, C-03, F-01, M-01. Fixes focus traps, Escape-to-close, consistent styling, and z-index management for the most critical user flow.

### 2. Establish i18n Infrastructure and Unify Language
Addresses C-01, I-01, I-02. Adopt `svelte-i18n`, extract all strings to locale files, and pick a single default language.

### 3. Add Mobile Cart Persistence (Floating CTA)
Addresses R-01, R-02. On mobile/tablet, always show a sticky floating bar with cart item count and total amount.

### 4. Improve Form Validation UX with Inline Errors
Addresses E-01, E-02, F-03. Replace toast-only validation with inline field-level error messages and auto-focus.

### 5. Add WCAG-Compliant Focus Indicators
Addresses A-06, A-07, A-08, A-09, CC-01. Ensure all interactive elements have visible focus indicators with sufficient contrast.
