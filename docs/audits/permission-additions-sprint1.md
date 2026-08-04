# Sprint 1 Item 3 — Permission Additions: `product.history.view` & `product.cost.view`

- **Status:** Approved (2026-08-04)
- **Backlog source:** `docs/audits/rbac-sprint0-audit.md` §15, item 3
- **Sync source:** `database` (permissions table) → `internal/permissions/permissions.go` (backend registry) → `web/src/shared/constants/permissions.ts` (frontend registry)

## Keputusan

Dua permission baru ditambahkan pada Sprint 1 Item 3, dengan pendekatan bertahap:

1. **Phase 3.1** — tambahkan permission ke DB + kedua registry, **tanpa perubahan behavior**.
2. **Phase 3.2** — `product.history.view` menggantikan compatibility layer role-based di UI.
3. **Phase 3.3** — `product.cost.view` memisahkan data sensitif (cost/margin) dari `pricing.view`, **dengan enforcement di backend** (bukan hanya sembunyi di UI).

## Matriks Permission

| Permission           | Default Grant            | Digunakan untuk                                                        |
| -------------------- | ------------------------ | ---------------------------------------------------------------------- |
| `product.history.view` | Superadmin, Admin        | Entity history: panel Audit Trail (created_at/updated_at) di product detail drawer |
| `product.cost.view`    | Superadmin, Admin, Manager | Data sensitif: cost, margin, purchase price, markup, profit            |

Alasan default grant:

- **`product.history.view` → SA/Admin:** menggantikan check role-based lama
  (`superadmin`/`admin`) persis tanpa perubahan perilaku.
- **`product.cost.view` → SA/Admin/Manager:** set yang sama dengan pemegang
  `pricing.view` saat ini, sehingga tidak ada regresi UX setelah pemisahan.
  Cashier & Staff tetap tidak melihat cost.

## Scope Konrol Setelah Pemisahan

**`product.cost.view` mengontrol:**
- cost (Harga Beli) di product detail drawer
- margin & margin %
- purchase price
- markup
- profit

**`pricing.view` tetap mengontrol (TIDAK berubah):**
- pricing rules (`/pricing-rules*`)
- supplier pricing (`/suppliers`)
- discount configuration

## Enforcement Backend

`product.cost.view` adalah data sensitif — **backend adalah source of truth**.
Enforcement dilakukan di layer presenter/DTO (bukan query):

- `GET /products` (list, termasuk batch `?ids=`)
- `GET /products/:id`

Jika caller tidak memiliki `product.cost.view`, field `cost` **dihilangkan**
(omitted, bukan `null`) dari response. Pemegang `product.cost.view` menerima
payload lengkap.

Endpoint lain diverifikasi tidak membocorkan cost:

| Endpoint | Gate | Hasil |
| -------- | ---- | ----- |
| `GET /products/options` | (bebas) | hanya id/sku/name — aman |
| `GET /products/search` (pricing) | `pricing.view` | hanya id/name/sku/price — aman |
| `GET /products` & `GET /products/:id` | `product.cost.view` | cost di-omit untuk non-holder |

POS menggunakan `GET /products` namun tidak merender field `cost` di UI
(`PosProduct` mendeklarasikannya, komponen tidak membacanya) — tidak ada
regresi pada alur kasir.

## Follow-up (di luar scope Item 3)

- **`sale` module:** `SaleItem.Cost int json:"cost,omitempty"` ikut dikembalikan
  di response sale detail/list. `sale.view` dipegang cashier, sehingga cashier
  masih bisa membaca cost via endpoint sale. Dikategorikan sebagai follow-up
  terpisah (bukan bagian dari product module) — belum diputuskan apakah perlu
  digate `product.cost.view`.

## Rollout & Migrasi

Migrasi `024_add_product_history_cost_permissions.sql` harus diaplikasikan
**sebelum** binary yang meng-enforce `product.cost.view` di-deploy. Sebelum
migrasi berjalan, tidak ada user yang memegang `product.cost.view`, sehingga
cost disembunyikan untuk semua orang (degradasi aman, tidak crash).
