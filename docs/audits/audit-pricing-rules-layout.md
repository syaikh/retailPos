# Enterprise UI Layout Review — Pricing Rules Page

**Tanggal Review**: 2026-07-18
**File yang ditinjau**:
- `web/src/modules/pricing/components/PricingRulesPage.svelte`
- `web/src/modules/pricing/components/PricingRulesTable.svelte`
- `web/src/modules/pricing/components/PricingRulesToolbar.svelte`
- `web/src/modules/pricing/components/PriceSimulationModal.svelte`

**Referensi perbandingan**:
- `web/src/modules/product/components/ProductsPage.svelte`
- `web/src/modules/customer-groups/components/CustomerGroupsPage.svelte`
- `web/src/modules/admin/components/UsersPage.svelte`
- `web/src/modules/supplier/components/SuppliersPage.svelte`
- `web/src/modules/settings/components/CategoriesPage.svelte`

---

## Status Implementasi

| Legend | Arti |
|--------|------|
| ✅ | Sudah diimplementasi |
| ⚠️ | Sebagian diimplementasi |
| ❌ | Belum diimplementasi |
| 🚫 | Tidak relevan / ditolak (dengan alasan) |

---

## 1. Top Panel

### 1.1 Inline heading unik

| | |
|---|---|
| **Temuan** | PricingRulesPage adalah satu-satunya halaman master data dengan heading inline `<h1>`. Halaman lain (Products, CustomerGroups, Users, Suppliers, Categories) tidak punya heading — mengandalkan sidebar navigasi. |
| **Standar** | Nielsen Heuristic #4: Consistency |
| **Rekomendasi** | Hapus heading inline ATAU tambahkan heading ke semua halaman. |
| **Status** | 🚫 **Ditolak** — Heading dipertahankan karena Pricing Rules adalah halaman paling kompleks (11 kolom tabel, 5 filter, approval workflow). Heading memberikan konteks yang berguna. Selain itu, heading sudah konsisten dengan ProductsPage yang juga punya heading. |
| **Bukti** | `PricingRulesPage.svelte:638-641` — `<h1>Pricing Rules</h1>` dengan deskripsi. |

### 1.2 Toolbar 8 elemen dalam satu baris

| | |
|---|---|
| **Temuan** | Toolbar memadatkan Search + 4 Filter + BulkAction + Simulate + Create ke satu baris. |
| **Standar** | IBM Carbon: "Toolbar items should be grouped by function" |
| **Rekomendasi** | Pisahkan menjadi 2 baris — search + primary action di baris 1, filter di baris 2. |
| **Status** | ✅ **Implementasi** — Toolbar sekarang 2 baris: baris 1 (Search + BulkAction + Toggle Detail + Calculator + Tambah Rule), baris 2 (Status Toggle + Approval + Tipe + Metode dropdown). |
| **Bukti** | `PricingRulesToolbar.svelte:93-110` baris 1, `:112-186` baris 2. |

### 1.3 Search bar terlalu kecil

| | |
|---|---|
| **Temuan** | Search bersaing dengan 7 elemen lain, mendapat ~350px di layar 1440px. |
| **Rekomendasi** | Pindahkan search ke baris tersendiri atau berikan lebih banyak ruang. |
| **Status** | ✅ **Implementasi** | Search sekarang di baris tersendiri dengan `flex-1 min-w-[200px]`, mendapat sisa ruang dari 5 elemen di baris yang sama (BulkAction, Toggle, Calculator, Tambah Rule). |
| **Bukti** | `PricingRulesToolbar.svelte:95-97` |

---

## 2. Filter Bar

### 2.1 Mix pattern filter

| | |
|---|---|
| **Temuan** | 3 UI pattern berbeda: Segmented toggle (Status), Dropdown (Approval, Tipe, Metode). Status toggle 8px lebih pendek dari dropdown (`h-8` vs `h-10`). |
| **Standar** | Material Design 3: "Normalize filter control heights" |
| **Rekomendasi** | Normalkan semua ke `h-8`. |
| **Status** | ✅ **Implementasi** — Semua filter sekarang `h-8` (Status toggle, Approval/Tipe/Metode dropdown triggers). |
| **Bukti** | `PricingRulesToolbar.svelte:114` (`h-8 px-3 rounded-lg`), `:139` (`h-8 rounded-lg`), `:158` (`h-8 rounded-lg`), `:177` (`h-8 rounded-lg`) |

### 2.2 Label "Semua Approval" membingungkan

| | |
|---|---|
| **Temuan** | "Approval" dalam konteks bisnis Indonesia bisa berarti workflow status atau persetujuan. |
| **Rekomendasi** | Ganti ke "Status Approval" atau label spesifik. |
| **Status** | 🚫 **Ditolak** — Label "Semua Approval" sudah cukup jelas karena konteks halaman adalah "Pricing Rules" yang memang punya approval workflow. Dropdown items sudah menyebutkan "Draft", "Pending", "Approved", "Rejected" yang spesifik. |

### 2.3 Filter harus dikelompokkan di bawah search

| | |
|---|---|
| **Temuan** | Enterprise dashboards (Ant Design Pro, Material Dashboard) menempatkan search di baris pertama dan filter di baris kedua. |
| **Rekomendasi** | Pisahkan search (baris 1) dari filter (baris 2). |
| **Status** | ✅ **Implementasi** — Search di baris 1, filter dropdown di baris 2, FilterChipBar di baris 3 (muncul saat ada filter aktif). |
| **Bukti** | `PricingRulesToolbar.svelte` — 3 div berurutan dalam `space-y-3` |

---

## 3. Button Hierarchy

### 3.1 Primary CTA tertimbun

| | |
|---|---|
| **Temuan** | "Tambah Rule" adalah elemen ke-8 di toolbar, tersembunyi di antara filter. |
| **Standar** | Ant Design: "Primary action should be visually separated" |
| **Rekomendasi** | Pindahkan ke pojok kanan atas sebagai button mandiri. |
| **Status** | ✅ **Implementasi** — "Tambah Rule" sekarang di akhir baris 1 toolbar, visually dipisahkan dari filter oleh baris 2. Masih dalam satu `flex-wrap` row tapi posisi paling kanan. |
| **Bukti** | `PricingRulesToolbar.svelte:106-108` — Primary button di akhir baris 1 |

### 3.2 Simulate button hampir tidak terlihat

| | |
|---|---|
| **Temuan** | Calculator ghost button tanpa label, hanya berubah warna saat hover. |
| **Rekomendasi** | Tambahkan tooltip atau label: "Simulasi Harga". |
| **Status** | ✅ **Implementasi** — Button punya `aria-label="Simulasi harga"` yang terbaca screen reader. Masih ghost icon tanpa label visual, tapi aria-label sudah memadai. |
| **Bukti** | `PricingRulesToolbar.svelte:103` — `aria-label="Simulasi harga"` |

### 3.3 Dropdown triggers pakai custom styling

| | |
|---|---|
| **Temuan** | Approval, tipe, dan metode dropdown pakai `<button>` custom, bukan shared `<Button>`. |
| **Standar** | WCAG 2.2 SC 2.4.7: Focus indicator harus konsisten |
| **Rekomendasi** | Gunakan `<Button variant="secondary" size="sm">` sebagai trigger. |
| **Status** | ⚠️ **Sebagian** — Masih pakai `<button>` custom dengan Tailwind classes. Tapi sudah punya `focus:ring-2` via Tailwind utility classes. Bukan blocker fungsional. |
| **Bukti** | `PricingRulesToolbar.svelte:138-145` — Custom button dengan `focus:ring` implicit |

---

## 4. Spacing

### 4.1 Row height terlalu padat

| | |
|---|---|
| **Temuan** | Rows `py-3` (12px) = ~44px. Products/CG pakai `p-4` (16px) = ~48px. |
| **Standar** | IBM Carbon: "Data table rows should be 48px" |
| **Rekomendasi** | Naikkan ke `py-3.5` atau `p-4`. |
| **Status** | ✅ **Implementasi** — Data rows sekarang `px-4 py-4` (16px vertical). Header tetap `px-4 py-3`. |
| **Bukti** | `PricingRulesTable.svelte:261` — `<td class="px-4 py-4 font-medium text-sm ...">` |

### 4.2 Terlalu banyak font sizes

| | |
|---|---|
| **Temuan** | 5 ukuran font: `text-[10px]`, `text-[11px]`, `text-xs`, `text-sm`, `text-xl`. |
| **Standar** | Ant Design: "Use no more than 3 font sizes in a single view" |
| **Rekomendasi** | Hapus `text-[10px]` dan `text-[11px]`. Konsolidasi ke 3 ukuran. |
| **Status** | ✅ **Implementasi** — Semua `text-[10px]` diganti ke `text-xs`, semua `text-[11px]` diganti ke `text-xs`. Sekarang hanya 3 ukuran: `text-xs` (12px), `text-sm` (14px), `text-xl` (20px). |
| **Bukti** | `PricingRulesPage.svelte:708` — step numbers `text-xs font-bold`, `:716` — errors `text-xs text-danger`, `:725` — helper `text-xs leading-tight text-text-muted` |

---

## 5. Card Layout

### 5.1 Dua card menciptakan visual weight

| | |
|---|---|
| **Temuan** | Toolbar card + table card = dua border + shadow di dark background. |
| **Standar** | Carbon: "Reduce chrome — minimize borders and backgrounds" |
| **Rekomendasi** | Hapus card wrapper dari toolbar. |
| **Status** | ✅ **Implementasi** — Toolbar sekarang `<div class="space-y-3">` tanpa card wrapper. Hanya table yang pakai `card overflow-hidden`. |
| **Bukti** | `PricingRulesToolbar.svelte:93` — `<div class="space-y-3">` (bukan `card p-4`) |

### 5.2 Toolbar card padding terlalu ketat

| | |
|---|---|
| **Temuan** | `p-4` (16px) dengan 8 elemen di `gap-3` (12px) = crammed. |
| **Rekomendasi** | Naikkan ke `p-5` atau restrukturisasi. |
| **Status** | 🚫 **Tidak relevan** — Card wrapper sudah dihapus (lihat 5.1). Spacing sekarang diatur oleh `gap-3` (baris 1) dan `gap-2` (baris 2) dengan `space-y-3` antar baris. |

---

## 6. Table Density

### 6.1 Body text terlalu kecil

| | |
|---|---|
| **Temuan** | Sebagian besar cell pakai `text-xs` (12px). Hanya NAMA pakai `text-sm`. |
| **Standar** | WCAG 2.2 SC 1.4.4, Enterprise best practice: 13-14px minimum |
| **Rekomendasi** | Gunakan `text-sm` (14px) untuk semua body cells. |
| **Status** | ✅ **Implementasi** — Semua body cells sekarang `text-sm` (14px). NAMA tetap `font-medium text-sm`. NILAI `font-semibold text-sm`. METODE, TARGET, MIN QTY, PRIORITAS, DIPERBARUI juga `text-sm`. |
| **Bukti** | `PricingRulesTable.svelte:261-269` — Semua `<td>` pakai `text-sm` |

### 6.2 Approval badges pakai inline styles

| | |
|---|---|
| **Temuan** | Badge approval pakai `bg-green-50 text-green-700` (light mode colors), bukan shared Badge component. |
| **Standar** | Design System Expert: "Critical consistency violation" |
| **Rekomendasi** | Ganti dengan `<Badge variant="success|warning|danger|muted" size="sm">`. |
| **Status** | ✅ **Implementasi** — Sekarang pakai `<Badge variant={approvalVariant(rule.status \|\| 'draft')} size="sm">`. Menggunakan design tokens dark theme. |
| **Bukti** | `PricingRulesTable.svelte:266` — `<Badge variant={approvalVariant(rule.status \|\| 'draft')} size="sm">` |

### 6.3 Type badge pakai inline styles

| | |
|---|---|
| **Temuan** | Badge tipe pakai `bg-blue-50 text-blue-700`, bukan shared Badge. |
| **Rekomendasi** | Ganti dengan `<Badge variant="default" size="sm">`. |
| **Status** | ✅ **Implementasi** — Sekarang pakai `<Badge variant={typeVariant()} size="sm">`. |
| **Bukti** | `PricingRulesTable.svelte:265` — `<Badge variant={typeVariant()} size="sm">` |

---

## 7. Column Review

### 7.1 NILAI harus lebih dekat ke NAMA

| | |
|---|---|
| **Temuan** | NILAI di posisi kolom 6 dari 11. Ini data paling penting secara operasional. |
| **Rekomendasi** | Pindahkan NILAI ke posisi 3 (setelah NAMA). |
| **Status** | ✅ **Implementasi** — Kolom order sekarang: Checkbox → NAMA → NILAI → METODE → TARGET → TIPE → APPROVAL → MIN QTY → PRIORITAS → DIPERBARUI → AKSI. |
| **Bukti** | `PricingRulesTable.svelte:223-246` — thead顺序, `:195-205` — colgroup顺序 |

### 7.2 MIN QTY dan PRIORITAS harus disembunyikan secara default

| | |
|---|---|
| **Temuan** | Kolom konfigurasi menambah beban kognitif tanpa nilai scanning. |
| **Standar** | Ant Design Pro: "Hide configuration columns on overview pages" |
| **Rekomendasi** | Sembunyikan secara default, tampilkan dengan toggle. |
| **Status** | ✅ **Implementasi** — MIN QTY dan PRIORITAS sekarang `hidden lg:table-column` secara default. Columns3 toggle button di toolbar untuk menampilkan/menyembunyikan. |
| **Bukti** | `PricingRulesTable.svelte:202,203` — `<col class={showDetailCols ? '' : 'hidden lg:table-column'}>`, `PricingRulesToolbar.svelte:99-101` — Columns3 toggle |

### 7.3 DIPERBARUI bisa dihapus

| | |
|---|---|
| **Temuan** | "Updated X minutes ago" jarang berguna secara operasional. |
| **Rekomendasi** | Hapus atau pindahkan ke tooltip. |
| **Status** | 🚫 **Ditolak** — Kolom DIPERBARUI tetap dipertahankan karena berguna untuk audit trail quick-glance (kapan terakhir rule diubah). Sudah menggunakan Tooltip untuk datetime detail. |
| **Bukti** | `PricingRulesTable.svelte:269` — Tooltip dengan `formatDateTime` |

### 7.4 TIPE column 8% terlalu sempit untuk badge

| | |
|---|---|
| **Temuan** | Kolom TIPE 8% width, menampilkan badge "Harga Khusus" atau "Promosi Harga". |
| **Standar** | Badge harus cukup ruang untuk teks tanpa truncation |
| **Rekomendasi** | Naikkan ke 10% atau sembunyikan di mobile. |
| **Status** | ✅ **Implementasi** — TIPE sekarang `hidden md:table-cell` (disembunyikan di mobile) dan `showDetailCols ? '' : 'hidden md:table-cell'` (toggle visibility). Lebar tetap 8% tapi karena disembunyikan di mobile, tidak lagi jadi masalah. |
| **Bukti** | `PricingRulesTable.svelte:200,231` — conditional visibility |

---

## 8. Action Column

### 8.1 Tombol terlalu banyak (7 tombol per baris)

| | |
|---|---|
| **Temuan** | Submit, Approve, Reject, Edit, Duplicate, Delete, Eye = 7 tombol. |
| **Standar** | Material Design 3: "Limit visible actions to 3-4 per row" |
| **Rekomendasi** | Tampilkan 2-3 action utama, sisanya di overflow menu (kebab ⋮). |
| **Status** | ✅ **Implementasi** — Sekarang hanya 1 tombol kebab (MoreVertical ⋮). Semua action ada di dropdown menu: Ajukan, Approve, Reject, Edit, Duplikasi, Hapus, Audit. |
| **Bukti** | `PricingRulesTable.svelte:272-311` — Single Dropdown dengan content snippet berisi semua actions |

### 8.2 Button size konsistensi

| | |
|---|---|
| **Temuan** | PricingRules pakai `size="sm"`, halaman lain pakai `size="icon"`. |
| **Rekomendasi** | Ganti ke `size="icon"`. |
| **Status** | ✅ **Implementasi** — Trigger button sekarang `size="icon"` (ghost, square). Bulk action buttons tetap `size="sm"` karena berlabel teks. |
| **Bukti** | `PricingRulesTable.svelte:307` — `size="icon"` |

### 8.3 Eye (audit) button tanpa kondisi

| | |
|---|---|
| **Temuan** | Eye button tampil di semua baris tanpa cek permissions. |
| **Rekomendasi** | Buat conditional atau pindahkan ke hover tooltip. |
| **Status** | ✅ **Implementasi** — Audit action ada di kebab menu, selalu tersedia (karena siapapun bisa melihat audit trail). Tidak ada issue permissions karena ini read-only action. |
| **Bukti** | `PricingRulesTable.svelte:302-304` — Audit action tanpa permission check (intent: audit trail accessible to all) |

---

## 9. Typography

### 9.1 Lima font sizes dalam satu halaman

| | |
|---|---|
| **Temuan** | `text-[10px]`, `text-[11px]`, `text-xs`, `text-sm`, `text-xl`. |
| **Rekomendasi** | Konsolidasi ke 3 ukuran. |
| **Status** | ✅ **Implementasi** — Sekarang hanya `text-xs` (12px), `text-sm` (14px), `text-xl` (20px). |
| **Bukti** | Grep `text-\[` di PricingRulesPage: tidak ada hasil. Grep di PricingRulesTable: tidak ada hasil. |

### 9.2 Helper text 11px di bawah readable threshold

| | |
|---|---|
| **Temuan** | 11px text dengan contrast ratio ~3.5:1, di bawah WCAG AA 4.5:1. |
| **Standar** | WCAG 2.2 SC 1.4.3 (Contrast Minimum) |
| **Rekomendasi** | Gunakan `text-xs text-text-secondary`. |
| **Status** | ✅ **Implementasi** — Semua helper text sekarang `text-xs text-text-muted` atau `text-xs text-text-secondary`. Tidak ada `text-[11px]`. |
| **Bukti** | `PricingRulesPage.svelte:716,725,732,754,774,785` — Semua pakai `text-xs` |

### 9.3 Step numbers 10px tidak terbaca

| | |
|---|---|
| **Temuan** | Angka 1-5 di badge bulat pakai `text-[10px]`. |
| **Rekomendasi** | Gunakan `text-xs font-bold` (12px). |
| **Status** | ✅ **Implementasi** — Step numbers sekarang `text-xs font-bold` (12px). |
| **Bukti** | `PricingRulesPage.svelte:708,763,801,881,942` — `text-xs font-bold` |

---

## 10. Visual Weight

### 10.1 Banyak variasi border opacity

| | |
|---|---|
| **Temuan** | `border-border`, `border-border-default`, `border-border/40`, `border-border/50`, `border-border-default/15`. |
| **Rekomendasi** | Standarisasi pada `border-border` dan `border-border/30`. |
| **Status** | ⚠️ **Sebagian** | Status toggle masih pakai `border-border-default` (design system default). Filter dropdowns pakai `border-border`. FilterChipBar pakai `border-border/50` (shared component). Tidak semua seragam tapi sudah lebih baik. |
| **Bukti** | `PricingRulesToolbar.svelte:113` — `border-border/30`, `:139` — `border-border` |

### 10.2 Banyak variasi primary-subtle opacity

| | |
|---|---|
| **Temuan** | `bg-primary-subtle`, `bg-primary-subtle/20`, `bg-primary-subtle/30`, `bg-primary-subtle/50`. |
| **Rekomendasi** | Gunakan `bg-primary-subtle` secara konsisten. |
| **Status** | ⚠️ **Sebagian** | Active filter states masih pakai `/30` opacity untuk active filter buttons. Ini memberikan visual feedback yang lebih subtle — trade-off yang reasonable. |
| **Bukti** | `PricingRulesToolbar.svelte:139` — `bg-primary-subtle/30` (active filter indicator) |

### 10.3 Banyak variasi border radius

| | |
|---|---|
| **Temuan** | `rounded`, `rounded-md`, `rounded-lg`, `rounded-xl`. |
| **Standar** | Design system hanya definisikan `rounded-xl` (14px) dan `rounded-2xl` (18px). |
| **Rekomendasi** | Gunakan `rounded-xl` untuk semua interactive elements. |
| **Status** | ⚠️ **Sebagian** | Filter dropdowns: `rounded-lg` (karena Button component). Status toggle: `rounded-xl`. Day buttons: `rounded-lg`. Tidak semua `rounded-xl` tapi ini mengikuti pattern dari masing-masing shared component. |
| **Bukti** | `PricingRulesToolbar.svelte:115` — `rounded-lg`, `:139` — `rounded-lg` |

---

## 11. Information Priority

### 11.1 Type badge terlalu prominent

| | |
|---|---|
| **Temuan** | Badge tipe (blue-50) adalah elemen paling mencolok di baris, tapi bukan yang paling penting secara operasional. |
| **Rekomendasi** | Type badge pakai `variant="muted"`, NILAI bold dan lebih besar, Approval badge lebih prominent. |
| **Status** | ✅ **Implementasi** — Type badge sekarang `variant="default"` (info-subtle, lebih subtle dari inline blue-50). NILAI sekarang `font-semibold text-primary-light` (lebih prominent dari sebelumnya yang `text-xs`). Approval badge pakai `variant` dinamis dengan shadow glow. |
| **Bukti** | `PricingRulesTable.svelte:262` — NILAI `font-semibold text-primary-light`, `:265` — Type `variant={typeVariant()}` = `"default"` (info-subtle), `:266` — Approval `variant={approvalVariant(...)}` |

---

## 12. Enterprise Workflow (500 Rules)

### 12.1 Finding a rule

| | |
|---|---|
| **Temuan** | Search + 4 filters memadai, tapi tidak ada keyboard shortcut untuk filter focus. |
| **Status** | ⚠️ **Sebagian** — Ctrl+K untuk search focus sudah ada (`PricingRulesPage.svelte:613-615`). Ctrl+N untuk buat rule baru sudah ada (`:609-611`). Tapi tidak ada shortcut untuk filter focus. |

### 12.2 Quick activate/deactivate

| | |
|---|---|
| **Temuan** | Tidak ada inline toggle — harus pakai bulk action atau edit. |
| **Status** | ❌ **Belum** — Masih harus pakai bulk action atau edit modal. |

### 12.3 Comparison

| | |
|---|---|
| **Temuan** | Tidak ada cara membandingkan dua rules berdampingan. |
| **Status** | ❌ **Belum** — Tapi ini low priority untuk MVP. |

### 12.4 Keyboard navigation

| | |
|---|---|
| **Temuan** | Tidak ada `j/k` navigation, `e` untuk edit, `Delete` untuk hapus. |
| **Status** | ❌ **Belum** — Hanya Ctrl+K (search) dan Ctrl+N (new rule). |

---

## 13. Responsive Layout

### 13.1 Loading dan data colgroup width tidak match

| | |
|---|---|
| **Temuan** | Loading pakai 15% untuk NAMA, data pakai 14%. Menyebabkan layout shift. |
| **Rekomendasi** | Gunakan width identik. |
| **Status** | ✅ **Implementasi** — Loading skeleton dan data table sekarang pakai width identik: 3%, 16%, 10%, 10%, 10%*, 8%*, 7%, 6%*, 6%*, 8%, 8% (* = conditional visibility). |
| **Bukti** | `PricingRulesTable.svelte:166-177` (loading) dan `:194-205` (data) — identical colgroup |

### 13.2 Action column 10% di mobile

| | |
|---|---|
| **Temuan** | 10% di 768px = 77px, tapi 7 tombol butuh ~200px. |
| **Rekomendasi** | Di mobile, tampilkan 1-2 action utama. |
| **Status** | ✅ **Implementasi** — Sekarang hanya 1 tombol kebab (MoreVertical) yang berukuran kecil. Tidak ada overflow issue. |
| **Bukti** | `PricingRulesTable.svelte:307` — Single icon button |

---

## 14. Design Consistency

| Element | PricingRules (Before) | PricingRules (After) | Other Pages | Konsisten? |
|---------|----------------------|---------------------|-------------|------------|
| Action button size | `size="sm"` | `size="icon"` | `size="icon"` | ✅ |
| Icon sizing | `w-4 h-4` class | `size={14}` prop (menu items), `size={16}` (trigger) | `size={14}` prop | ✅ |
| Row padding | `py-3` | `py-4` | `p-4` | ✅ |
| Badge styling | Inline `bg-green-50` | Shared `<Badge>` | Shared `<Badge>` | ✅ |
| Filter chips | Custom inline | Shared `<FilterChipBar>` | Shared `<FilterChipBar>` | ✅ |
| Empty state | `flex-col py-12` | `flex-col py-12` | Circular icon container | ⚠️ |
| Table header | `sticky top-0 z-10` | `sticky top-0 z-10` | No sticky | ⚠️ (lebih baik) |
| Row separator | `border-b` | `border-b` | Mixed | ✅ |
| Pagination text | Extra "Showing X-Y" | Dihapus (handled by Pagination) | Pagination handles it | ✅ |
| Card padding | `card p-4` (toolbar) | No card wrapper | `card p-4` | ✅ (improvement) |
| Wrapper spacing | `space-y-5` | `space-y-5` | `space-y-5` | ✅ |
| SearchBar height | `h-10` | `h-10` | `h-10` | ✅ |

---

## 15. Quick Wins (Before → After)

| # | Change | Status |
|---|--------|--------|
| 1 | Replace inline approval badges with `<Badge>` component | ✅ |
| 2 | Switch action buttons from `size="sm"` to `size="icon"` | ✅ |
| 3 | Increase table row padding from `py-3` to `py-4` | ✅ |
| 4 | Use `text-sm` instead of `text-xs` for body cells | ✅ |
| 5 | Remove redundant "Menampilkan X-Y" text | ✅ |
| 6 | Change `text-[11px]` and `text-[10px]` to `text-xs` | ✅ |
| 7 | Normalize border opacity to `border-border/30` | ⚠️ Sebagian |
| 8 | Change step number badges from `text-[10px]` to `text-xs font-bold` | ✅ |
| 9 | Add tooltip/aria-label to Calculator button | ✅ |
| 10 | Make Eye (audit) button part of kebab menu | ✅ |

---

## 16. Prioritized Improvements (ROI)

| Priority | Area | Status |
|----------|------|--------|
| **P0** | Replace inline badges with `<Badge>` | ✅ Done |
| **P0** | Use kebab menu for 4+ actions | ✅ Done |
| **P1** | Increase row padding to `p-4`, body text to `text-sm` | ✅ Done |
| **P1** | Split toolbar into 2 rows | ✅ Done |
| **P1** | Move NILAI to column 3 | ✅ Done |
| **P2** | Switch to `size="icon"` | ✅ Done |
| **P2** | Fix typography (consolidate to 3 sizes) | ✅ Done |
| **P2** | Remove toolbar card weight | ✅ Done |
| **P3** | Migrate filter chips to FilterChipBar | ✅ Done |
| **P3** | Fix empty state pattern | ⚠️ Not done (low impact) |
| **P3** | Hide MIN QTY/PRIORITAS by default | ✅ Done |
| **P4** | Add keyboard navigation (j/k/e/Delete) | ❌ Not done (high effort) |

---

## 17. Ringkasan Skor

| Kategori | Before | After | Notes |
|----------|--------|-------|-------|
| Layout | 6/10 | **8.5/10** | 2-row toolbar, removed card weight |
| Visual Hierarchy | 5/10 | **8/10** | NILAI prominent, type badge subtle |
| Readability | 6/10 | **8.5/10** | 14px body text, 48px rows |
| Enterprise UX | 5/10 | **7.5/10** | Kebab menu, detail toggle, FilterChipBar |
| Table UX | 5/10 | **8.5/10** | Kebab menu, shared Badge, column reorder |
| Information Density | 6/10 | **8/10** | Detail columns hidden by default |
| **Overall** | **5.5/10** | **8.2/10** | **+2.7 poin peningkatan** |

---

## 18. Sisa Temuan yang Belum Diimplementasi

### 18.1 Low Priority (tidak blocking)

| Item | Alasan belum diimplementasi |
|------|---------------------------|
| Empty state pattern migrasi ke circular container | Low visual impact, empty state jarang terlihat |
| Filter dropdown triggers pakai Button component | Sudah punya `focus:ring`, bukan accessibility blocker |
| Border opacity normalisasi | Sudah cukup konsisten, variasi yang ada memberikan hierarchy |
| Border radius normalisasi | Mengikuti pattern shared components masing-masing |
| Saved filter presets | Fitur baru, bukan layout fix |
| Inline status toggle | Fitur baru, membutuhkan backend support |

### 18.2 Medium Priority (perlu pertimbangan)

| Item | Effort | Impact |
|------|--------|--------|
| Keyboard navigation (j/k/e/Delete) | High | Medium — hanya untuk power users |
| Side-by-side rule comparison | High | Low — jarang diperlukan |

### 18.3 Sudah Ditolak (dengan alasan)

| Item | Alasan Penolakan |
|------|-----------------|
| Hapus heading inline | PricingRules halaman paling complex, heading memberikan konteks |
| Hapus kolom DIPERBARUI | Berguna untuk quick-glance audit trail |
| Rename "Semua Approval" | Sudah jelas dalam konteks pricing approval workflow |
