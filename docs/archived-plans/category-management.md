# Plan: Halaman Manajemen Kategori (Revised)

## Overview
Implementasi modul Category Management (Halaman Manajemen Kategori) untuk Retail POS System, mencakup backend API (Go/Gin), frontend UI (Svelte 5), RBAC permissions, restricted delete logic, slug auto-generation, dan audit logging.

**Revisions from initial plan** (based on review feedback):
1. Race condition protection: Change FK `products.category_id` from `ON DELETE SET NULL` → `ON DELETE RESTRICT` + app-level check with `EXISTS` (early exit)
2. Slug collision: Implement iterative suffix `-2`, `-3` etc. in `CreateCategory` using SELECT EXISTS loop
3. Domain model separation: Create `internal/domain/category.go` instead of appending to `user.go`
4. Svelte 5 reactivity: Use `oninput` event binding instead of `$effect` for search
5. Delete button: Simplify `onclick` (remove inline guard since `disabled` already blocks clicks)
6. is_active toggle: Add toggle in Edit modal form
7. Query optimization: Use `LEFT JOIN + GROUP BY` instead of correlated subquery for `GetAllCategories`
8. Partial index: Add `idx_products_category_deleted` on `products(category_id) WHERE deleted_at IS NULL`
9. Delete validation: Use `EXISTS` instead of `COUNT(*)` for `HasActiveProducts`
10. Audit old_values: Ensure `old` snapshot is fetched from DB before applying request changes

---

## 1. Database Migration

### File: `database/migrations/008_category_management.sql`

```sql
-- Migration: 008_category_management.sql
-- Description: Add slug, updated_at to categories; change FK to RESTRICT; add category management perms
-- Created: 2026-06-03

-- Add slug and updated_at columns
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS slug VARCHAR(120) UNIQUE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Backfill slug for existing categories
UPDATE categories SET slug = LOWER(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(TRIM(name), ' ', '-'), '''', ''), '"', ''), '&', 'and'), '/', '-'))
WHERE slug IS NULL;

-- Change FK constraint: products.category_id ON DELETE RESTRICT (prevent delete if products exist)
-- Step 1: Drop existing FK constraint
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_category_id_fkey;
-- Step 2: Re-add with RESTRICT
ALTER TABLE products ADD CONSTRAINT products_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

-- Partial index: covers the JOIN condition in GetAllCategories and the EXISTS check
-- This makes COUNT/EXISTS on products with category_id + deleted_at IS NULL instant
CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id) WHERE deleted_at IS NULL;

-- New permissions for category management
INSERT INTO permissions (code, name, description)
VALUES
    ('category.update', 'Edit Kategori', 'Bisa mengubah kategori'),
    ('category.delete', 'Hapus Kategori', 'Bisa menghapus kategori')
ON CONFLICT (code) DO NOTHING;

-- Grant to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('category.update', 'category.delete')
WHERE r.name = 'superadmin'
ON CONFLICT DO NOTHING;

-- Grant to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('category.update', 'category.delete')
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;
```

**Key design decisions:**
- FK changed to `ON DELETE RESTRICT` — database itself rejects delete when products reference the category (race condition safe)
- Partial index `idx_products_category_active` covers `(category_id) WHERE deleted_at IS NULL` — optimizes both the LEFT JOIN in list queries and the EXISTS validation
- App-level `EXISTS` check provides user-friendly error message; DB-level RESTRICT is the safety net

---

## 2. Domain Model

### New File: `internal/domain/category.go`

Separate file following Clean Architecture — Category is a bounded context, not a user concern.

```go
package domain

type Category struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Slug         string `json:"slug,omitempty"`
    Description  string `json:"description,omitempty"`
    IsActive     bool   `json:"is_active"`
    ProductCount int    `json:"product_count,omitempty"`
    CreatedAt    string `json:"created_at,omitempty"`
    UpdatedAt    string `json:"updated_at,omitempty"`
}

type CategoryCreateRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}

type CategoryUpdateRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    IsActive    *bool  `json:"is_active,omitempty"`
}
```

### File: `internal/domain/user.go`

**Remove** the existing `Category` struct (lines 101-108) from this file. All category-related types now live in `category.go`.

**Update `generateAuditDescription`** handler to import and handle `domain.Category` from the new file (same package, no import change needed since both are in `domain`).

---

## 3. Repository Layer

### File: `internal/repository/repository.go`

Add `CategoryRepository` interface:

```go
type CategoryRepository interface {
    GetAllCategories(ctx context.Context, limit, offset int, search string) ([]domain.Category, int, error)
    GetCategoryByID(ctx context.Context, id int) (*domain.Category, error)
    CreateCategory(ctx context.Context, category *domain.Category) error
    UpdateCategory(ctx context.Context, category *domain.Category) error
    DeleteCategory(ctx context.Context, id int) error
    HasActiveProducts(ctx context.Context, categoryID int) (bool, error)
    SlugExists(ctx context.Context, slug string, excludeID int) (bool, error)
}
```

**Key changes from initial plan:**
- `GetProductCountByCategory` → `HasActiveProducts` — returns `bool` (uses EXISTS, not COUNT)
- Added `SlugExists` — for iterative slug collision resolution (with `excludeID` for updates)

### File: `internal/repository/postgres_repository.go`

#### `GetAllCategories` — Optimized with LEFT JOIN + GROUP BY

```go
func (r *postgresRepository) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]domain.Category, int, error) {
    // COUNT query (no JOIN needed for counting)
    countQuery := `SELECT COUNT(*) FROM categories WHERE 1=1`
    args := []interface{}{}
    argIdx := 1
    if search != "" {
        countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx)
        args = append(args, "%"+search+"%")
        argIdx++
    }
    var total int
    if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("failed to count categories: %w", err)
    }

    // DATA query — LEFT JOIN + GROUP BY (single pass, no correlated subquery)
    query := `SELECT c.id, c.name, COALESCE(c.slug, ''), COALESCE(c.description, ''), c.is_active,
              COUNT(p.id) AS product_count,
              c.created_at, COALESCE(c.updated_at, c.created_at)
              FROM categories c
              LEFT JOIN products p ON p.category_id = c.id AND p.deleted_at IS NULL
              WHERE 1=1`
    args2 := []interface{}{}
    argIdx2 := 1
    if search != "" {
        query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.slug ILIKE $%d)", argIdx2, argIdx2)
        args2 = append(args2, "%"+search+"%")
        argIdx2++
    }
    query += " GROUP BY c.id"
    query += fmt.Sprintf(" ORDER BY c.name ASC LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
    args2 = append(args2, limit, offset)

    rows, err := r.db.Query(ctx, query, args2...)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to query categories: %w", err)
    }
    defer rows.Close()

    var categories []domain.Category
    for rows.Next() {
        var c domain.Category
        var createdAt, updatedAt time.Time
        if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &c.ProductCount, &createdAt, &updatedAt); err != nil {
            continue
        }
        c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
        c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
        categories = append(categories, c)
    }
    return categories, total, nil
}
```

**Why LEFT JOIN + GROUP BY is better than correlated subquery:**
- Correlated subquery (`SELECT COUNT(*) ... WHERE category_id = c.id`) runs N+1 times
- LEFT JOIN + GROUP BY processes in a single pass using the partial index `idx_products_category_active`

#### `GetCategoryByID`

```go
func (r *postgresRepository) GetCategoryByID(ctx context.Context, id int) (*domain.Category, error) {
    var c domain.Category
    var createdAt, updatedAt time.Time
    err := r.db.QueryRow(ctx, `
        SELECT id, name, COALESCE(slug, ''), COALESCE(description, ''), is_active,
               created_at, COALESCE(updated_at, created_at)
        FROM categories WHERE id = $1
    `, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt, &updatedAt)
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, fmt.Errorf("category not found")
        }
        return nil, err
    }
    c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
    c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
    return &c, nil
}
```

#### `CreateCategory` — with iterative slug collision resolution

```go
func (r *postgresRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
    if category.Slug == "" {
        category.Slug = generateSlug(category.Name)
    }

    // Resolve slug collision: append -2, -3, etc.
    baseSlug := category.Slug
    suffix := 1
    for {
        exists, err := r.SlugExists(ctx, category.Slug, 0)
        if err != nil {
            return fmt.Errorf("failed to check slug uniqueness: %w", err)
        }
        if !exists {
            break
        }
        suffix++
        category.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
        // Safety: limit slug length
        if len(category.Slug) > 120 {
            category.Slug = fmt.Sprintf("%s-%d", baseSlug[:120-len(fmt.Sprintf("-%d", suffix))], suffix)
        }
    }

    var createdAt, updatedAt time.Time
    err := r.db.QueryRow(ctx, `
        INSERT INTO categories (name, slug, description, is_active)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at
    `, category.Name, category.Slug, category.Description, category.IsActive).Scan(&category.ID, &createdAt, &updatedAt)
    if err != nil {
        return err
    }
    category.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
    category.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
    return nil
}
```

#### `UpdateCategory` — regenerate slug, handle collision with exclude

```go
func (r *postgresRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
    newSlug := generateSlug(category.Name)
    // Only resolve collision if slug changed
    if newSlug != category.Slug {
        exists, err := r.SlugExists(ctx, newSlug, category.ID)
        if err != nil {
            return fmt.Errorf("failed to check slug uniqueness: %w", err)
        }
        if exists {
            // Append suffix for collision
            suffix := 2
            for {
                candidate := fmt.Sprintf("%s-%d", newSlug, suffix)
                ex, err := r.SlugExists(ctx, candidate, category.ID)
                if err != nil {
                    return err
                }
                if !ex {
                    newSlug = candidate
                    break
                }
                suffix++
            }
        }
        category.Slug = newSlug
    }

    _, err := r.db.Exec(ctx, `
        UPDATE categories SET name = $1, slug = $2, description = $3, is_active = $4, updated_at = NOW()
        WHERE id = $5
    `, category.Name, category.Slug, category.Description, category.IsActive, category.ID)
    return err
}
```

#### `DeleteCategory`

```go
func (r *postgresRepository) DeleteCategory(ctx context.Context, id int) error {
    _, err := r.db.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
    return err
}
```

**Note:** If products still reference this category (race condition), PostgreSQL's `ON DELETE RESTRICT` FK constraint will reject the DELETE and return an error. The handler catches this and returns a user-friendly message.

#### `HasActiveProducts` — EXISTS with early exit (NOT COUNT)

```go
func (r *postgresRepository) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
    var exists bool
    err := r.db.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM products
            WHERE category_id = $1 AND deleted_at IS NULL
            LIMIT 1
        )
    `, categoryID).Scan(&exists)
    return exists, err
}
```

**Why EXISTS instead of COUNT:**
- `COUNT(*)` scans all matching rows
- `EXISTS` stops at the first match (early exit)
- For validation ("can I delete?"), we only need to know *if* products exist, not *how many*
- The partial index `idx_products_category_active` makes this essentially an index-only check

#### `SlugExists`

```go
func (r *postgresRepository) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
    var exists bool
    query := `SELECT EXISTS (SELECT 1 FROM categories WHERE slug = $1 AND id != $2)`
    err := r.db.QueryRow(ctx, query, slug, excludeID).Scan(&exists)
    return exists, err
}
```

#### `generateSlug` helper

```go
func generateSlug(name string) string {
    slug := strings.ToLower(strings.TrimSpace(name))
    replacements := []struct{ from, to string }{
        {" ", "-"}, {"'", ""}, {`"`, ""}, {"&", "and"}, {"/", "-"},
        {"+", "plus"}, {"=", "equals"}, {"?", ""}, {"!", ""}, {"@", "at"},
        {"#", "number"}, {"%", "percent"}, {"(", ""}, {")", ""},
    }
    for _, r := range replacements {
        slug = strings.ReplaceAll(slug, r.from, r.to)
    }
    for strings.Contains(slug, "--") {
        slug = strings.ReplaceAll(slug, "--", "-")
    }
    slug = strings.Trim(slug, "-")
    if len(slug) > 120 {
        slug = slug[:120]
    }
    return slug
}
```

---

## 4. Handler Layer

### File: `internal/delivery/http/handler/handler.go`

#### Update Handler struct

```go
type Handler struct {
    authRepo      repository.UserRepository
    roleRepo      repository.RoleRepository
    productRepo   repository.ProductRepository
    saleRepo      repository.SaleRepository
    authService   *auth.AuthService
    hub          *websocket.Hub
    auditRepo    repository.AuditLogRepository
    categoryRepo repository.CategoryRepository
}
```

#### Update NewHandler

```go
func NewHandler(
    authRepo repository.UserRepository,
    roleRepo repository.RoleRepository,
    productRepo repository.ProductRepository,
    saleRepo repository.SaleRepository,
    authService *auth.AuthService,
    hub *websocket.Hub,
    auditRepo repository.AuditLogRepository,
    categoryRepo repository.CategoryRepository,
) *Handler {
    return &Handler{
        authRepo:      authRepo,
        roleRepo:      roleRepo,
        productRepo:   productRepo,
        saleRepo:      saleRepo,
        authService:   authService,
        hub:          hub,
        auditRepo:    auditRepo,
        categoryRepo: categoryRepo,
    }
}
```

#### `canManageCategory` helper

```go
func (h *Handler) canManageCategory(c *gin.Context, permission string) bool {
    role := h.userRole(c)
    if role == "superadmin" || role == "admin" {
        return true
    }
    return h.hasPermission(c, permission)
}
```

#### `ListCategoriesManagement`

```go
func (h *Handler) ListCategoriesManagement(c *gin.Context) {
    role := h.userRole(c)
    if role == "cashier" {
        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
        return
    }
    if !h.hasPermission(c, "category.view") {
        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
        return
    }

    limit := 20
    if l := c.Query("limit"); l != "" {
        if val, err := strconv.Atoi(l); err == nil && val > 0 {
            limit = val
        }
    }
    offset := 0
    if o := c.Query("offset"); o != "" {
        if val, err := strconv.Atoi(o); err == nil && val >= 0 {
            offset = val
        }
    }
    search := c.Query("search")

    categories, total, err := h.categoryRepo.GetAllCategories(getCtx(c), limit, offset, search)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch categories"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": categories, "total": total})
}
```

#### `CreateCategoryHandler`

```go
func (h *Handler) CreateCategoryHandler(c *gin.Context) {
    if !h.canManageCategory(c, "category.create") {
        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
        return
    }

    var req domain.CategoryCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    if strings.TrimSpace(req.Name) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
        return
    }

    category := &domain.Category{
        Name:        strings.TrimSpace(req.Name),
        Description: strings.TrimSpace(req.Description),
        IsActive:    true,
    }

    if err := h.categoryRepo.CreateCategory(getCtx(c), category); err != nil {
        if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "violate") {
            c.JSON(http.StatusConflict, gin.H{"error": "category name or slug already exists"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category"})
        return
    }
    h.logAudit(c, "create", "category", category.ID, nil, category)
    c.JSON(http.StatusCreated, gin.H{"data": category})
}
```

#### `UpdateCategoryHandler` — fetches old from DB BEFORE applying changes for audit

```go
func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
    if !h.canManageCategory(c, "category.update") {
        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    // Fetch OLD state from DB for accurate audit logging
    old, err := h.categoryRepo.GetCategoryByID(getCtx(c), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
        return
    }

    var req domain.CategoryUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    if strings.TrimSpace(req.Name) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
        return
    }

    // Apply changes to a copy for update (keep old intact for audit)
    updated := *old
    updated.Name = strings.TrimSpace(req.Name)
    updated.Description = strings.TrimSpace(req.Description)
    if req.IsActive != nil {
        updated.IsActive = *req.IsActive
    }

    if err := h.categoryRepo.UpdateCategory(getCtx(c), &updated); err != nil {
        if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "violate") {
            c.JSON(http.StatusConflict, gin.H{"error": "category name or slug already exists"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update category"})
        return
    }
    // old = pure DB state before mutation; updated = state after mutation
    h.logAudit(c, "update", "category", updated.ID, old, updated)
    c.JSON(http.StatusOK, gin.H{"data": updated})
}
```

**Critical:** `old` is fetched from DB *before* any request body is applied, ensuring `old_values` in audit log is the true prior state.

#### `DeleteCategoryHandler` — dual protection (app-level EXISTS + DB-level RESTRICT)

```go
func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
    if !h.canManageCategory(c, "category.delete") {
        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))
    old, err := h.categoryRepo.GetCategoryByID(getCtx(c), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
        return
    }

    // Application-level check: provides user-friendly error message
    hasProducts, err := h.categoryRepo.HasActiveProducts(getCtx(c), id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check product association"})
        return
    }
    if hasProducts {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Gagal menghapus! Kategori masih digunakan oleh produk aktif.",
        })
        return
    }

    // DB-level RESTRICT FK is the safety net for race conditions
    if err := h.categoryRepo.DeleteCategory(getCtx(c), id); err != nil {
        // If RESTRICT constraint kicks in (race condition), return same friendly message
        if strings.Contains(err.Error(), "restrict") || strings.Contains(err.Error(), "violates foreign key") {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Gagal menghapus! Kategori masih digunakan oleh produk aktif.",
            })
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete category"})
        return
    }
    h.logAudit(c, "delete", "category", id, old, nil)
    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
```

**Defense in depth:**
1. `HasActiveProducts` (EXISTS query) — fast, user-friendly error
2. `ON DELETE RESTRICT` FK — database-enforced safety net if a product is assigned between the check and the DELETE

#### `generateAuditDescription` update

```go
// Add to the switch statement inside getIdentifier:
case *domain.Category:
    return v.Name
case domain.Category:
    return v.Name
```

---

## 5. Routes

### File: `cmd/server/main.go`

```go
// Add categoryRepo (reuses same db pool)
categoryRepo := repository.NewPostgresRepository(dbPool)

// Update NewHandler call
h := handler.NewHandler(authRepo, roleRepo, productRepo, saleRepo, authService, hub, auditRepo, categoryRepo)

// Add routes under protected group:
protected.GET("/categories/manage", h.ListCategoriesManagement)
protected.POST("/categories", h.CreateCategoryHandler)
protected.PUT("/categories/:id", h.UpdateCategoryHandler)
protected.DELETE("/categories/:id", h.DeleteCategoryHandler)
```

The existing `GET /categories` public route (for product form dropdown) remains unchanged.

---

## 6. Frontend

### File: `web/src/lib/types.ts`

Update `Category` interface:

```typescript
export interface Category {
  id: number;
  name: string;
  slug?: string;
  description?: string;
  is_active?: boolean;
  product_count?: number;
  created_at?: string;
  updated_at?: string;
}
```

### File: `web/src/lib/pages/admin/Categories.svelte`

Key revisions from initial plan:
- **Search reactivity**: Use `oninput` event handler instead of `$effect` watching `searchQuery`
- **Delete button**: Simplified `onclick` — `disabled` attribute already prevents clicks
- **is_active toggle**: Added in Edit modal
- **form state**: Includes `is_active` for edit mode

```svelte
<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth } from '$lib/stores/auth';

  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, Plus, Pencil, Trash2, Tag, Loader2, X } from 'lucide-svelte';

  let loading = $state(true);
  let categories = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedCategory = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let isInitialMount = $state(true);

  let prevOffset = 0;
  let prevLimit = 20;

  let form = $state({
    name: '',
    description: '',
    is_active: true
  });

  // RBAC
  let userRole = $derived(
    $auth.user?.role?.name ||
    ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) ||
    ''
  );
  let canCreate = $derived(['superadmin', 'admin'].includes(userRole));
  let canEdit = $derived(['superadmin', 'admin'].includes(userRole));
  let canDelete = $derived(['superadmin', 'admin'].includes(userRole));
  let canView = $derived(userRole !== 'cashier');

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    const d = new Date(dateStr);
    const months = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'];
    return `${String(d.getDate()).padStart(2,'0')} ${months[d.getMonth()]} ${d.getFullYear()}`;
  }

  async function fetchCategories(isSearch = false) {
    if (!canView) return;
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      const res = await apiFetch(`/api/categories/manage?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        categories = data.data || [];
        total = data.total || 0;
      } else if (res.status === 403) {
        toast.error('Akses ditolak');
        categories = [];
        total = 0;
      }
    } catch {
      toast.error('Gagal memuat kategori');
    } finally {
      if (!isSearch) loading = false;
    }
  }

  onMount(async () => {
    if (!canView) { loading = false; return; }
    isInitialMount = true;
    await fetchCategories(false);
    isInitialMount = false;
  });

  // Search: event-driven, NOT $effect
  function handleSearchInput() {
    offset = 0;
    prevOffset = 0;
    if (searchQuery === '') {
      fetchCategories(false);
    } else {
      debouncedSearchFetch();
    }
  }

  const debouncedSearchFetch = debounce(() => {
    fetchCategories(true);
  }, 400);

  function clearSearch() {
    searchQuery = '';
    offset = 0;
    prevOffset = 0;
    fetchCategories(false);
  }

  // Pagination: event-driven via onPageChange callback (no $effect needed)
  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    prevOffset = newOffset;
    prevLimit = newLimit;
    fetchCategories(false);
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', description: '', is_active: true };
    showModal = true;
  }

  function openEdit(cat) {
    modalMode = 'edit';
    selectedCategory = cat;
    form = {
      name: cat.name,
      description: cat.description || '',
      is_active: cat.is_active !== false
    };
    showModal = true;
  }

  function openDelete(cat) {
    selectedCategory = cat;
    showDeleteModal = true;
  }

  async function saveCategory() {
    if (!form.name.trim()) {
      toast.error('Nama kategori wajib diisi');
      return;
    }
    try {
      saving = true;
      const method = modalMode === 'add' ? 'POST' : 'PUT';
      const url = modalMode === 'add' ? '/api/categories' : `/api/categories/${selectedCategory.id}`;
      const payload = {
        name: form.name.trim(),
        description: form.description.trim()
      };
      // Only send is_active on update
      if (modalMode === 'edit') {
        payload.is_active = form.is_active;
      }
      const r = await apiFetch(url, { method, body: JSON.stringify(payload) });
      if (r.ok) {
        toast.success(modalMode === 'add' ? 'Kategori berhasil ditambahkan' : 'Kategori berhasil diperbarui');
        showModal = false;
        await fetchCategories();
      } else {
        const err = await r.json();
        toast.error(err.error || 'Gagal menyimpan kategori');
      }
    } catch {
      toast.error('Kesalahan jaringan');
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUser) return;
    try {
      const r = await apiFetch(`/api/categories/${selectedCategory.id}`, { method: 'DELETE' });
      if (r.ok) {
        toast.success(`Kategori "${selectedCategory.name}" berhasil dihapus`);
        await fetchCategories();
      } else {
        const err = await r.json();
        toast.error(err.error || 'Gagal menghapus kategori');
      }
    } catch {
      toast.error('Gagal menghapus kategori');
    } finally {
      showDeleteModal = false;
      selectedCategory = null;
    }
  }
</script>

{#if !canView}
  <div class="flex flex-col items-center justify-center py-20">
    <div class="w-20 h-20 rounded-2xl bg-danger-subtle flex items-center justify-center mb-4">
      <Tag size={32} class="text-danger" />
    </div>
    <h2 class="text-xl font-bold text-text-primary mb-2">Akses Ditolak</h2>
    <p class="text-text-muted">Anda tidak memiliki izin untuk mengakses halaman ini.</p>
  </div>
{:else}
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-text-primary">Manajemen Kategori</h2>
        <p class="text-text-muted">Kelola kategori produk toko Anda</p>
      </div>
      <div class="flex items-center gap-2">
        {#if canCreate}
          <button class="btn btn-primary" onclick={openAdd}>
            <Plus size={16} /> Tambah Kategori
          </button>
        {/if}
      </div>
    </div>

    <!-- Search -->
    <div class="card p-4">
      <div class="relative max-w-sm">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <!-- oninput triggers search directly — no $effect needed -->
        <input
          type="text"
          placeholder="Cari nama kategori…"
          class="input pl-9 pr-10"
          bind:value={searchQuery}
          oninput={handleSearchInput}
        />
        {#if searchQuery}
          <button
            onclick={clearSearch}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Hapus pencarian"
          >
            <X size={14} />
          </button>
        {/if}
      </div>
    </div>

    <!-- Table -->
    <div class="card p-0 overflow-hidden">
      <div class="px-4 py-3 border-b border-border flex items-center justify-between">
        <p class="text-sm font-semibold text-text-primary">Daftar Kategori</p>
        {#if !loading}
          <span class="badge badge-muted">{total} kategori</span>
        {/if}
      </div>

      {#if loading}
        <div class="divide-y divide-border">
          {#each { length: 5 } as _}
            <div class="flex items-center gap-4 px-4 py-3.5">
              <Skeleton width="w-40" height="h-3.5" />
              <Skeleton width="w-28" height="h-3.5" />
              <Skeleton width="w-12" height="h-6" rounded="rounded-full" />
              <Skeleton width="w-20" height="h-3" />
            </div>
          {/each}
        </div>
      {:else if categories.length === 0}
        <div class="px-4 py-12 text-center">
          <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
            <Tag size={32} class="text-text-muted" />
          </div>
          <p class="text-text-primary font-semibold mt-4">Tidak ada kategori</p>
          <p class="text-text-muted text-sm mt-1">
            {searchQuery ? `Tidak ditemukan untuk "${searchQuery}"` : 'Mulai dengan menambahkan kategori'}
          </p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table>
            <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
              <tr>
                <th>Nama Kategori</th>
                <th>Slug</th>
                <th class="text-center">Jumlah Produk</th>
                <th>Tanggal Dibuat</th>
                <th class="text-center">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {#each categories as cat (cat.id)}
                <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                  <td>
                    <div class="flex items-center gap-3">
                      <div class="w-9 h-9 rounded-xl bg-primary-subtle flex items-center justify-center shrink-0">
                        <Tag size={14} class="text-primary-light" />
                      </div>
                      <div>
                        <p class="font-medium text-text-primary">{cat.name}</p>
                        {#if cat.description}
                          <p class="text-xs text-text-muted truncate max-w-[200px]">{cat.description}</p>
                        {/if}
                      </div>
                    </div>
                  </td>
                  <td>
                    <code class="text-xs bg-surface-default px-2 py-1 rounded text-text-muted">{cat.slug}</code>
                  </td>
                  <td class="text-center">
                    <span class="inline-flex items-center justify-center min-w-[28px] px-2 py-0.5 rounded-full text-xs font-semibold
                      {cat.product_count > 0 ? 'bg-primary-subtle text-primary-light' : 'bg-surface-default text-text-muted'}">
                      {cat.product_count}
                    </span>
                  </td>
                  <td class="text-text-secondary text-sm">
                    {formatDate(cat.created_at)}
                  </td>
                  <td>
                    <div class="flex items-center justify-center gap-2">
                      {#if canEdit}
                        <button
                          class="btn-icon btn-ghost text-text-muted hover:text-primary-light"
                          title="Edit"
                          onclick={() => openEdit(cat)}
                        >
                          <Pencil size={14} />
                        </button>
                      {/if}
                      {#if canDelete}
                        <button
                          class="btn-icon btn-ghost {cat.product_count > 0 ? 'text-text-muted/30 cursor-not-allowed' : 'text-text-muted hover:text-danger hover:bg-danger-subtle'}"
                          onclick={() => openDelete(cat)}
                          disabled={cat.product_count > 0}
                          title={cat.product_count > 0 ? 'Tidak bisa dihapus: masih ada produk aktif' : 'Hapus'}
                        >
                          <Trash2 size={14} />
                        </button>
                      {/if}
                      {#if !canEdit && !canDelete}
                        <span class="text-xs text-text-muted">—</span>
                      {/if}
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="p-4 bg-surface-subtle/30">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Add/Edit Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Tambah Kategori' : 'Edit Kategori'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveCategory(); }} class="space-y-4">
    <div>
      <label for="cat-name" class="block text-sm font-medium text-text-secondary mb-2">Nama Kategori <span class="text-danger">*</span></label>
      <input id="cat-name" type="text" placeholder="Contoh: Makanan Bayi" class="input" bind:value={form.name} required />
    </div>
    <div>
      <label for="cat-desc" class="block text-sm font-medium text-text-secondary mb-2">Deskripsi <span class="text-text-muted text-xs">(opsional)</span></label>
      <textarea id="cat-desc" placeholder="Deskripsi singkat kategori…" class="input min-h-[80px] resize-y" bind:value={form.description}></textarea>
    </div>
    <!-- is_active toggle: only shown in edit mode (new categories always active) -->
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-3 cursor-pointer select-none group">
          <div class="relative">
            <input type="checkbox" class="sr-only peer" bind:checked={form.is_active} />
            <div class="w-10 h-5 bg-surface-default border border-border rounded-full peer peer-checked:bg-primary-subtle peer-checked:border-primary/50 transition-colors"></div>
            <div class="absolute left-1 top-1 w-3 h-3 bg-text-muted rounded-full peer-checked:translate-x-5 peer-checked:bg-primary-light transition-transform shadow-sm"></div>
          </div>
          <span class="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">
            {form.is_active ? 'Aktif' : 'Tidak Aktif'}
          </span>
        </label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showModal = false} disabled={saving}>Batal</button>
    <button class="btn btn-primary min-w-32" onclick={saveCategory} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Menyimpan...
      {:else}
        {modalMode === 'add' ? 'Tambah Kategori' : 'Simpan Perubahan'}
      {/if}
    </button>
  {/snippet}
</Modal>

<!-- Delete Confirm Modal -->
<Modal bind:open={showDeleteModal} title="Hapus Kategori" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Hapus "{selectedCategory?.name}"?</p>
    <p class="text-text-muted text-sm">Kategori ini akan dihapus secara permanen dan tidak dapat dikembalikan.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>Batal</button>
    <button class="btn btn-danger" onclick={confirmDelete}>Hapus</button>
  {/snippet}
</Modal>
```

**Svelte 5 reactivity improvements:**
- Search uses `oninput={handleSearchInput}` — no `$effect` watching `searchQuery`
- Pagination uses `onPageChange` callback directly — no `$effect` watching `offset/limit`
- This avoids infinite loops and makes data flow explicit

**Delete button simplification:**
- `disabled={cat.product_count > 0}` already prevents click events in the browser
- `onclick={() => openDelete(cat)}` is clean — no inline guard needed

---

## 7. Frontend: Router & Sidebar

### File: `web/src/lib/App.svelte`

```svelte
import AdminCategories from '$lib/pages/admin/Categories.svelte';

// In getComponent:
case '/admin/categories': return AdminCategories;
```

### File: `web/src/lib/components/Sidebar.svelte`

```svelte
import { Tag } from 'lucide-svelte';

// In adminItems array:
{ label: 'Categories', href: '/admin/categories', icon: Tag },
```

---

## 8. RBAC Summary

| Role | View | Create | Edit | Delete |
|------|------|--------|------|--------|
| Superadmin | ✅ | ✅ | ✅ | ✅ (restricted) |
| Admin | ✅ | ✅ | ✅ | ✅ (restricted) |
| Manager | ✅ (view only) | ❌ | ❌ | ❌ |
| Cashier | ❌ (403) | ❌ | ❌ | ❌ |

**Enforcement layers:**
- Frontend: Conditionally hides/shows buttons based on `userRole`
- Backend: Permission check on every handler (`canManageCategory` + `hasPermission`)
- Database: `ON DELETE RESTRICT` FK constraint (immutable safety net)

---

## 9. Audit Log

All mutations logged via `h.logAudit()`:
- `entity_type`: `"category"`
- `action`: `"create"`, `"update"`, `"delete"`
- `old_values`: Pure DB state fetched *before* mutation (for update/delete)
- `new_values`: Post-mutation state (for create/update)
- `generateAuditDescription` updated: `"Created category: Makanan Bayi"`, `"Updated category: Makanan Bayi"`, etc.

---

## 10. Files to Create/Modify

### New Files:
1. `database/migrations/008_category_management.sql`
2. `internal/domain/category.go`
3. `web/src/lib/pages/admin/Categories.svelte`

### Modified Files:
1. `internal/domain/user.go` — Remove `Category` struct (moved to `category.go`)
2. `internal/repository/repository.go` — Add `CategoryRepository` interface
3. `internal/repository/postgres_repository.go` — Add `CategoryRepository` impl, `generateSlug`, `SlugExists`
4. `internal/delivery/http/handler/handler.go` — Add category handlers, update `Handler` struct/NewHandler, update `generateAuditDescription`
5. `cmd/server/main.go` — Add `categoryRepo` + routes
6. `web/src/lib/App.svelte` — Add route + import
7. `web/src/lib/components/Sidebar.svelte` — Add Categories nav item
8. `web/src/lib/types.ts` — Update `Category` interface

---

## 11. Implementation Order

1. `database/migrations/008_category_management.sql`
2. `internal/domain/category.go` (new) + remove Category from `user.go`
3. `internal/repository/repository.go` — add interface
4. `internal/repository/postgres_repository.go` — implement methods
5. `internal/delivery/http/handler/handler.go` — update Handler, add handlers
6. `cmd/server/main.go` — wire up repo + routes
7. `go build ./...` — verify compilation
8. `web/src/lib/types.ts` — update Category interface
9. `web/src/lib/pages/admin/Categories.svelte` — new page
10. `web/src/lib/App.svelte` — add route
11. `web/src/lib/components/Sidebar.svelte` — add nav item
12. Run migration on dev DB, test API endpoints, test frontend
