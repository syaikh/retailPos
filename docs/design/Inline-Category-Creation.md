# Inline Category Creation in Product Form

**Status:** Proposed
**Date:** 2026-09-04
**Scope:** Product form modal, product service, backend category resolution

---

## 1. Problem

The category field in the Add/Edit Product modal is a text input with a dropdown that searches existing categories. When a user types a category name that doesn't exist:

1. **Current broken behavior:** The frontend sends `category_name: "Organic Food"` but never sends `category_id`. The backend receives `CategoryName` but `CategoryID` stays `nil`. The product is created with `category_id = NULL` — even for **existing** categories.

2. **No inline creation:** Users must leave the product form, navigate to Category Management, create the category, return to the product form, and re-enter all data. This is a poor UX for a common workflow.

3. **Data quality risk:** Without inline creation, staff either skip the category field (NULL) or use inconsistent naming (`"Food"`, `"food"`, `"FOOD "`) because they can't create a proper category on the spot.

---

## 2. Decision

**Add a "Create 'X'" option at the bottom of the category dropdown when no exact match exists.**

Clicking it creates the category via `POST /categories`, adds it to the local list, and auto-selects it. Backend also gets a safety-net `CategoryName → CategoryID` resolution.

---

## 3. Current State

| Layer | Category Handling |
|-------|------------------|
| Database | `categories.name VARCHAR(100) UNIQUE` — uniqueness enforced |
| Backend `POST /categories` | Creates category (requires `category.create` permission) |
| Backend `CreateProduct` | Uses only `CategoryID` — no `CategoryName` resolution |
| Backend `CategoryRepo.GetCategoryIDByName` | Exists, used only in list/filter path |
| Frontend `ProductFormModal` | Text input + dropdown, filters `categories: string[]` prop |
| Frontend `ProductsPage` | Sends `category_name: form.category` in payload, no `category_id` |
| RBAC | `category.create` permission: superadmin, admin, manager (NOT cashier) |

---

## 4. Process Flow

```
User types "Organic Food" in category field
            │
            ▼
   ┌─────────────────────────┐
   │ Search existing cats    │
   │ matching "Organic Food" │
   └────────┬────────────────┘
            │
    ┌───────┴───────┐
    │ Exact match?  │
    └───┬───────┬───┘
        │       │
       YES      NO
        │       │
        ▼       ▼
   Show      Show filtered
   matches   matches + "Create 'Organic Food'"
                │
                ▼
        User clicks "Create"
                │
                ▼
        POST /categories { name: "Organic Food" }
                │
        ┌───────┴───────┐
        │               │
      201 OK        Error (e.g. 409
        │           duplicate name)
        ▼               ▼
   Add to local    Show error toast,
   categories      keep dropdown open
   list, select it,
   show success toast
                │
                ▼
        User submits product form
                │
                ▼
        POST /products { category_name: "Organic Food", ... }
                │
                ▼
        Backend resolves CategoryName → CategoryID (safety net)
                │
                ▼
        Product created with correct category_id
```

---

## 5. Mockup

### State 1: No match — "Create" option visible

```
┌─────────────────────────────────────────────┐
│  Category *                                 │
│  ┌─────────────────────────────────────┐    │
│  │ Organic Food                    ↵  │    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │  No categories found               │    │
│  ├─────────────────────────────────────┤    │
│  │ ✚  Create "Organic Food"           │ ◀──│── Clickable
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### State 2: Partial matches — "Create" option at bottom

```
┌─────────────────────────────────────────────┐
│  Category *                                 │
│  ┌─────────────────────────────────────┐    │
│  │ Food                            ↵  │    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │  Food & Beverages                  │    │
│  │  Food Court                        │    │
│  ├─────────────────────────────────────┤    │
│  │ ✚  Create "Food"                   │ ◀──│── Clickable
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### State 3: Exact match — NO "Create" option

```
┌─────────────────────────────────────────────┐
│  Category *                                 │
│  ┌─────────────────────────────────────┐    │
│  │ Food & Beverages                ↵  │    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │  Food & Beverages                  │ ◀──│── Exact match, selected
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### State 4: Creating — loading spinner

```
┌─────────────────────────────────────────────┐
│  Category *                                 │
│  ┌─────────────────────────────────────┐    │
│  │ Organic Food                    ↵  │    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │ ⟳  Creating "Organic Food"...      │ ◀──│── Disabled, spinner
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### State 5: No permission — "Create" option hidden

```
┌─────────────────────────────────────────────┐
│  Category *                                 │
│  ┌─────────────────────────────────────┐    │
│  │ Organic Food                    ↵  │    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │  No categories found               │    │
│  └─────────────────────────────────────┘    │
│  (No "Create" option — user lacks           │
│   category.create permission)               │
└─────────────────────────────────────────────┘
```

---

## 6. Implementation Plan

### 6.1 Frontend — `product-service.ts`

Add `createCategory` function:

```typescript
export async function createCategory(name: string): Promise<Category> {
  const r = await apiClient.post('/categories', { name });
  return r.data.data;
}
```

### 6.2 Frontend — `ProductFormModal.svelte`

#### New imports

```typescript
import { createCategory } from '../services/product-service';
import { toast } from '$shared/stores/toast.svelte';
import { Loader2 } from 'lucide-svelte';
```

#### New state

```typescript
let creatingCategory = $state(false);
let localCategories = $state<string[]>([]);
```

#### New derived

```typescript
const canCreateCategory = rbac.can(Permissions.category.create);

const allCategories = $derived([
  ...categories,
  ...localCategories.filter(c => !categories.includes(c))
]);

const exactCategoryMatch = $derived(
  allCategories.some(c => c.toLowerCase() === modalCategorySearch.toLowerCase())
);

const showCreateOption = $derived(
  canCreateCategory &&
  modalCategorySearch.trim() !== '' &&
  !exactCategoryMatch &&
  !creatingCategory
);
```

#### New handler

```typescript
async function handleCreateCategory() {
  const name = modalCategorySearch.trim();
  if (!name) return;

  creatingCategory = true;
  try {
    const newCat = await createCategory(name);
    localCategories = [...localCategories, newCat.name];
    form.category = newCat.name;
    modalCategorySearch = newCat.name;
    showModalCategoryDropdown = false;
    toast.success(labels.toastCategoryAdded);
  } catch (err: any) {
    const msg = err.response?.data?.error || labels.toastFailedSaveCategory;
    toast.error(msg);
  } finally {
    creatingCategory = false;
  }
}
```

#### Update `filteredModalCategories`

Change from `categories` to `allCategories`:

```typescript
let filteredModalCategories = $derived(
  allCategories.filter(cat =>
    cat !== 'All' && cat.toLowerCase().includes(modalCategorySearch.toLowerCase())
  )
);
```

#### Update dropdown markup

After the `{#if filteredModalCategories.length === 0}` block and the `{#each}` loop, add:

```svelte
{#if showCreateOption}
  <div class="border-t border-border my-0.5"></div>
  <button
    type="button"
    onmousedown|preventDefault={() => handleCreateCategory()}
    class="flex items-center gap-2 px-3 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-xl transition-all duration-200 w-full text-left"
    disabled={creatingCategory}
  >
    {#if creatingCategory}
      <Loader2 size={14} class="animate-spin" />
      <span>{labels.creatingCategory}</span>
    {:else}
      <Plus size={14} />
      <span>{t('createCategoryInline', { name: modalCategorySearch.trim() })}</span>
    {/if}
  </button>
{/if}
```

#### Update dropdown max-height

The dropdown currently has `max-h-48`. With the new "Create" option, increase to `max-h-56` to accommodate the extra item without scrolling issues.

### 6.3 Frontend — i18n Labels

**File:** `web/src/shared/i18n/en.ts`

```typescript
createCategoryInline: 'Create "{name}"',
creatingCategory: 'Creating category...',
```

**File:** `web/src/shared/i18n/id.ts`

```typescript
createCategoryInline: 'Buat "{name}"',
creatingCategory: 'Membuat kategori...',
```

### 6.4 Backend — `internal/product/service.go` (Safety Net)

Enhance `CreateProduct` to resolve `CategoryName → CategoryID` when `CategoryID` is nil:

```go
func (s *service) CreateProduct(ctx context.Context, product *Product) error {
    if err := s.resolveCategoryID(ctx, product); err != nil {
        return err
    }
    return s.repo.CreateProduct(ctx, product)
}

func (s *service) UpdateProduct(ctx context.Context, product *Product) error {
    if _, err := s.repo.GetProductByID(ctx, product.ID, product.StoreID); err != nil {
        return err
    }
    if err := s.resolveCategoryID(ctx, product); err != nil {
        return err
    }
    if err := s.repo.UpdateProduct(ctx, product, product.StoreID); err != nil {
        return err
    }
    return s.eventBus.Publish(ctx, events.TopicProductUpdated, &events.ProductUpdated{
        ID:      product.ID,
        SKU:     product.SKU,
        Stock:   product.Stock,
    })
}

func (s *service) resolveCategoryID(ctx context.Context, product *Product) error {
    if product.CategoryID != nil || product.CategoryName == nil || *product.CategoryName == "" {
        return nil
    }
    id, err := s.categoryRepo.GetCategoryIDByName(ctx, *product.CategoryName)
    if err != nil {
        return fmt.Errorf("category %q not found", *product.CategoryName)
    }
    product.CategoryID = &id
    return nil
}
```

---

## 7. Validation Rules Summary

| Rule | Frontend | Backend | Rationale |
|------|----------|---------|-----------|
| Category required | `!form.category.trim()` | Not enforced (allows NULL for backward compat) | UX guard |
| Category exists | "Create" option + backend resolution | `GetCategoryIDByName` returns error if not found | Defense-in-depth |
| Permission to create | `rbac.can(Permissions.category.create)` | `category.create` permission check | RBAC enforcement |
| Duplicate name | N/A (caught by DB `UNIQUE(name)`) | Returns 409 or generic error | DB constraint |

---

## 8. Edge Cases

| Scenario | Behavior |
|----------|----------|
| User types category, clicks "Create", then cancels product | Category is created but product is not. Orphaned category is acceptable — can be cleaned up in Category Management |
| User types duplicate category name | Backend returns error from `POST /categories` (DB UNIQUE constraint). Toast shows error. Dropdown stays open |
| User lacks `category.create` permission | "Create" option is hidden. User can only select from existing categories |
| User types exact match of existing category | "Create" option is hidden. Dropdown shows the matching category for selection |
| User clicks "Create" while loading | Button is disabled during creation. No double-submit |
| Backend receives `category_name` that doesn't exist | `resolveCategoryID` returns error: `category "X" not found`. Product creation fails with clear message |
| User edits product and changes category to a new name | Same "Create" option appears in dropdown. Flow is identical to add mode |

---

## 9. Migration Considerations

- **No schema change** — `categories.name UNIQUE` already exists
- **No data migration** — existing products with NULL `category_id` remain unchanged
- **Backend safety net** — `resolveCategoryID` is backward compatible; existing products with valid `category_id` are unaffected

---

## 10. Testing Checklist

- [ ] Category dropdown shows "Create 'X'" when no exact match exists
- [ ] "Create" option is hidden when user lacks `category.create` permission
- [ ] Clicking "Create" calls `POST /categories` and shows loading spinner
- [ ] On success: new category appears in dropdown, is auto-selected, success toast shown
- [ ] On error (duplicate name): error toast shown, dropdown stays open
- [ ] Product form submits with `category_name` in payload
- [ ] Backend resolves `category_name` to `category_id` during product creation
- [ ] Backend returns error if `category_name` doesn't exist (safety net)
- [ ] Edit mode: same "Create" option works for changing category
- [ ] Existing categories still appear in dropdown and can be selected normally
