# Restrukturisasi i18n — Catatan Progres

Dokumen ini mencatat progres restrukturisasi internasionalisasi (i18n) pada frontend Svelte.
Perbarui setelah setiap sesi kerja.

Tanggal terakhir diperbarui: 2026-08-09

## Status Ringkas

| Item | Status |
|------|--------|
| Fondasi i18n (store + diksionari) | ✅ Selesai |
| Wiring `shared/ui` | ✅ Selesai |
| Wiring `shared` + `app` | ✅ Selesai |
| Wiring `modules/pos` | ✅ Selesai |
| Wiring `modules/product` | ✅ Selesai |
| Review agent + perbaikan temuan | ✅ Selesai |
| Wiring `modules/sales` | ✅ Selesai |
| Wiring `modules/*` lainnya | ⬜ Belum |
| Verifikasi akhir (vitest + build) | ⬜ Belum |

## Konteks & Keputusan

- Sasaran: menghilangkan string literal UI di komponen `.svelte`, diganti referensi
  `labels.*` dari store i18n, sehingga UI bisa beralih Indonesia/Inggris.
- Keputusan sesi ini (disetujui user):
  1. **Update test per komponen** — setiap `.svelte` yang di-wire ke `labels.*`
     sekaligus diupdate test source-structure guard-nya.
  2. **Wiring incremental module-by-module** — mulai dari `shared/ui` (reuse
     tertinggi), lalu `shared`+`app`, lalu tiap modul dengan test hijau per langkah.

## Fondasi (Selesai)

File: `web/src/shared/i18n/`

- `id.ts` — diksionari bahasa Indonesia, **kanonik** (~784 key, termasuk ~59 key
  produk dari sesi wiring).
- `en.ts` — mirror penuh bahasa Inggris (~784 key). Diverifikasi oleh test
  `en mirrors id keys exactly`.
- `index.svelte.ts` — store reaktif Svelte 5 runes (class `I18nStore`):
  - `i18n.locale` (`$state<Locale>`), default `'id'`, dipersistenkan ke
    `localStorage` key `pos.locale`.
  - `i18n.labels` (getter) — diksionari aktif.
  - `labels` — **Proxy** yang membungkus store agar `labels.save` dst. tetap
    reaktif terhadap perubahan locale (bukan snapshot).
  - Ekspor utilitas: `currentLocale()`, `setLocale(locale)`, `toggleLocale()`.
- Test: `web/src/shared/i18n/__tests__/i18n.test.ts` — **5/5 lolos**:
  default id, switch ke en + persist localStorage, toggle, reaktivitas labels,
  mirror id↔en.

### Catatan implementasi (perangkap yang sudah diatasi)

1. Svelte 5 melarang `export const x = $state(...)` jika di-reassign —
   solusinya pola class store (`new I18nStore()`) dengan getter `labels`.
2. `export const labels = i18n.labels` adalah **snapshot non-reaktif** —
   solusinya `new Proxy(i18n, get...)` yang tetap melacak `$state`.

## Temuan Penting — Hambatan Test

- **65 file** `.svelte.test.ts` adalah *source-structure guards*: membaca file
  source dan melakukan `expect(src).toContain('<literal UI>')`.
- Total **1.544 assertion** bertipe `toContain(...)` / `getByText` /
  `toHaveTextContent` di 65 file tersebut.
- Konsekuensi: mengganti string literal komponen dengan `{labels.xxx}` memecah
  assertion ini satu-per-satu → setiap wiring wajib diikuti update test-nya.

## Skala Yang Tersisa (terverifikasi sesi ini)

- **96 file** `.svelte` mengandung string literal UI (grep `>[^<]{*}...<`),
  di luar `__tests__` dan `.stories`.
- Estimasi audit awal: 135 file / 594 string literal (angka awal; terverifikasi
  ulang 96 file via grep berbeda — catat sebagai perbedaan metode deteksi).
- `shared/ui` dengan string literal (8 file):
  `PreviewTable`, `DropZone`, `Pagination`, `ImportWizard`, `ImportSummary`,
  `CurrencyInput`, `ConfirmDeleteModal`, `CashBreakdown`.
  (Contoh string: `"Rows per page:"` di `Pagination.svelte`.)
- Test `shared/ui` yang sudah ada (7 file): `CurrencyInput`, `Input`, `Modal`,
  `Pagination`, `SearchBar`, `StatCard`, `Toast`.

## Langkah Berikutnya

1. ✅ Wire `shared/ui` → 8 komponen ber-string ke `labels.*`, tambahkan key baru ke
   `id.ts`+`en.ts`, update test guard masing-masing, pastikan hijau.
2. ✅ Wire `shared` + `app` (layout, komponen umum non-module).
3. ✅ Wire `modules/pos` (lihat Catatan Sesi di bawah).
4. ✅ Wire `modules/product` (lihat Catatan Sesi di bawah).
5. ✅ Wire `modules/sales` (lihat Catatan Sesi di bawah).
6. Wire modul satu per satu (urutan reuse berikutnya).
7. Verifikasi akhir: `npx vitest run` penuh di `web/` + `npm run build`/svelte-check.

### Langkah modul `sales`

## Catatan Sesi 2026-08-09 (modules/sales)

File yang di-wire ke `labels.*` / `t()`:
- `TransactionTable.svelte` — aria-label loading (`labels.loadingTransactions`),
  empty state (`labels.noTransactionsFound`, `labels.tryAdjustingSearchOrDateRange`),
  header sortable (`labels.invoiceLabel/dateLabel/paymentLabel/totalRp`, plus
  `labels.customerLabel/itemsLabel`), customer fallback `labels.walkInGeneral`,
  `t('itemsCount', { count })`, `t('moreWithCount', { count })`.
- `TransactionFilters.svelte` — `datePresets` diubah dari const label-literal
  menjadi `$derived` dari `labels.today/yesterday/last7Days/last30Days/thisMonth/thisYear`;
  `dateRangeLabel` di-refactor agar match berdasarkan nilai `days` (bukan
  `p.label === 'Yesterday'`), custom `t('customDateRange', { start, end })`,
  fallback `labels.last30Days`; pesan error amount
  (`labels.errorMinCannotBeNegative`, `t('errorMinExceedsMax')`,
  `labels.errorMaxCannotBeNegative`, `t('errorMaxExceedsMax')`,
  `labels.errorMinCannotExceedMax`); placeholder `labels.minLabel/maxLabel`,
  `labels.filterJumlah`, `labels.allMethods`, `labels.clear/apply/cancel`,
  `labels.presetRanges/customRange/startDateLabel/endDateLabel`,
  `labels.exportCSV/exportExcel/export`, `labels.currencySymbol`,
  toast `labels.toastSessionExpired/toastExportFailed`.
- `TransactionDrawer.svelte` — heading/aria `labels.transactionDetails`,
  `labels.loading`, metadata (`invoiceNumber`, `dateAndTime`, `customer`,
  `paymentMethod`, `refLabel`), customer fallback `labels.walkInGeneral`,
  header tabel item (`labels.items/description/qty/price/subTotal`),
  `labels.hemat`, `{labels.subTotal} ({labels.dpp})`, `labels.ppn`,
  `labels.totalLabel`, `{labels.currencySymbol}` di total dan payment lines,
  tombol `labels.close/printReceipt/downloadInvoice`,
  toast `labels.toastInvoiceDownloaded/toastFailedToDownloadInvoice`.
- `TransactionsPage.svelte` — tidak mengandung literal UI; tidak perlu di-wire.

Key i18n baru (mirror id/en), kelompok `// ===== Transactions =====`:
`loadingTransactions`, `itemsLabel`, `moreWithCount`, `allMethods`,
`customDateRange`, `minLabel`, `maxLabel`, `startDateLabel`, `endDateLabel`,
`refLabel`, `downloadInvoice`, `errorMinCannotBeNegative`, `errorMinExceedsMax`,
`errorMaxCannotBeNegative`, `errorMaxExceedsMax`, `errorMinCannotExceedMax`,
`exportExcel`, `dateAndTime`, `toastSessionExpired`, `toastExportFailed`,
`toastInvoiceDownloaded`, `toastFailedToDownloadInvoice`.

Sengaja dipertahankan hardcoded: nilai data/status (`completed`, `refunded`,
`payment_method` codes), dash `'—'`, shortcut keyboard, dan nama file export
(`transactions-YYYY-MM-DD`).

Test: `npx vitest run src/shared/i18n src/modules/sales` → 126/126 hijau.
`svelte-check` penuh → hanya 3 error pre-existing (EditStorageLocationModal.svelte
dan useRBAC.test.ts).

## Catatan Sesi 2026-08-09 (modules/pos)

File yang di-wire ke `labels.*` / `t()`:
- `CartPanel.svelte` — header cart, empty state, tombol Hold/Clear, aria-label
  qty/remove, `priceFrozen`, tombol bayar `t('payWithAmount', { amount })`,
  Recall/Print (`t('recallWithCount')`, `t('printInvoice')`).
- `CheckoutModal.svelte` — judul/aria dialog, header tabel item, DPP/PPN, "Hemat"
  (`t('savingsWithAmount')`), `paymentAllocation`, remove-all/remove-payment,
  label Jumlah/No. Referensi, denom (`labels.denomMillion/Thousand`), `exactAmountShortcut`,
  `cancelEsc`, `doneEnter`, hint alokasi.
- `CustomerSelectModal.svelte` — heading, placeholder, `searching`, walk-in,
  `noPhone`, `customerNotFound`.
- `ParkedSalesModal.svelte` — heading, empty state, `t('cartWithId')`,
  `t('itemsCount')`, tombol Recall, prefix mata uang `labels.currencySymbol`.
- `PosPage.svelte` — fallback customer `labels.walkInGeneral`, seluruh toast
  (sukses/gagal cart, checkout, hold/resume, shift) via `labels.toast*`,
  prefix mata uang di receipt payload, toggle cart mobile (Hide/Show).
- `PosProductTable.svelte` — empty state, header tabel, copy SKU/barcode,
  tombol Add.
- `ProductSearchPanel.svelte` — placeholder pencarian.

Key i18n baru (mirror id/en), kelompok `// ===== POS =====`:
`hold`, `priceFrozen`, `priceFrozenTitle`, `processing`, `payWithAmount`,
`recall`, `recallWithCount`, `print`, `printInvoice`, `paymentAllocation`,
`removeAll`, `removeThisPayment`, `removePayment`, `amount`, `referenceNumber`,
`referenceNumberPlaceholder`, `denomMillion`, `denomThousand`,
`exactAmountShortcut`, `selectPaymentMethodHint`, `cancelEsc`, `doneEnter`,
`savingsWithAmount`, `searching`, `walkInGeneral`, `noPhone`, `cartWithId`,
`itemsCount`, `add`, `copySku`, `copyBarcode`, `searchByNameSkuBarcode`,
`currencySymbol`, `hide`, `show`, dan `toast*` (CartIsEmpty, SaleCompleted,
SaleHeld, SaleResumed, CartCleared, FailedToLoadProducts, FailedToAddItem,
FailedToRemoveItem, FailedToUpdateQuantity, FailedToClearCart,
FailedToHoldSale, FailedToResumeSale, CheckoutFailed, CheckoutFailedRetry,
MustOpenShiftFirst).

Sengaja dipertahankan hardcoded di POS: shortcut keyboard di dalam `<kbd>`
(F2/F5/F6/ALT+DEL) dan label fallback metode bayar `Cash`/`Card`/`E-Wallet`
(proper noun; hanya dipakai saat API payment-methods gagal).

Catatan: saat menambah key POS ternyata ada duplikat dengan blok dashboard
yang sudah dibuat sesi sebelumnya (`addProductsToStartSelling`, `yourCartIsEmpty`,
`holdSale`, `clearCart`, `decreaseQuantity`, `increaseQuantity`, `removeItem`,
`productName`, `noProductsFound`, `noHeldSales`, `searchByNameOrPhone`).
Duplikat dihapus; nilai lama yang dipertahankan. Key `exactAmount` (grup cashier
yang belum dipakai, 'Uang Pas') dibiarkan, key tombol F7 diberi nama
`exactAmountShortcut` agar tidak bentrok makna.

Test: `npx vitest run src/shared/i18n src/modules/pos` → 118/118 hijau.
`svelte-check` penuh → hanya 3 error pre-existing (EditStorageLocationModal.svelte
dan useRBAC.test.ts).

## Catatan Sesi 2026-08-09 (shared + app)

File yang di-wire ke `labels.*`:
- `app/main.svelte` — `pageTitles` diubah jadi `Record<string, () => string>` dan
  `updateTitle()` memanggil fungsi; label entitas impor-history pakai `labels.*`
  (tidak hardcode 'Kategori'/'Merek'/dst.); toast akses-ditolak pakai
  `labels.noPermissionToAccessPage`.
- `app/components/NotFoundPage.svelte` — `pageNotFound`, `pageNotFoundText`,
  `goBack`, `dashboard`.
- `app/components/ReceiptPrintOverlay.svelte` — label receipt (invoice, waktu,
  customer, dpp, ppn, total, payment, cash, receiptThanks, receiptNoReturn).
- `app/layouts/Sidebar.svelte` — semua item nav jadi `{ label: () => labels.x }`
  dirender `{item.label()}` (reaktif terhadap ganti locale); aria-label/title/
  section header pakai `labels.sidebar`, `mainNavigation`, `masterData`,
  `administration`, `logout`, `collapse(Sidebar)`, `expandSidebar`,
  `closeShiftFirst`, `managementSystem`.
- `app/layouts/NotificationBell.svelte` — markup dropdown pakai `labels.*`;
  handler WebSocket (12 event) pakai `t('key', { placeholder })` untuk
  title/deskripsi/toast stok rendah.

Key i18n baru ditambahkan di `id.ts`+`en.ts` (mirror):
- `noPermissionToAccessPage`, `pageNotFoundText`, `goBack`, `waktu`, `dpp`,
  `ppn`, `receiptThanks`, `receiptNoReturn`, `collapse`, `collapseSidebar`,
  `expandSidebar`, `closeShiftFirst`, `markAllRead`, `lowStockAlert`,
  `lowStockAlertDesc`, `lowStockAlertToast`, `newTransactionDesc`,
  `stockUpdated`, `stockUpdateStatusLow`, `stockUpdateUnits`, `stockUpdatedDesc`,
  `productUpdated`, `productUpdatedDesc`, `poReceived`, `poReceivedDesc`,
  `soCreatedTitle`, `soSessionCreatedDesc`, `soSubmittedTitle`,
  `soSessionAwaitingDesc`, `soApprovedTitle`, `soSessionApprovedDesc`,
  `soRejectedTitle`, `soSessionRejectedDesc`, `soNeedsRecountTitle`,
  `soSessionNeedsRecountDesc`, `soCancelledTitle`, `soSessionCancelledDesc`.

Sisa literal UI yang sengaja dipertahankan: nama brand (`RetailPOS`,
`RETAIL POS`) dan key data (`new_values`/`newValues`).

Test: i18n mirror + Sidebar guard → 17/17 hijau. `svelte-check` bersih untuk
file yang diubah (3 error tersisa adalah pre-existing di
`EditStorageLocationModal.svelte` dan `useRBAC.test.ts`).

## Catatan Sesi 2026-08-09 (modules/product)

File yang di-wire ke `labels.*` / `t()`:
- `ProductsPage.svelte` — toast seluruh operasi (import, bulk status, stok,
  add/update/delete, gagal muat master data) via `labels.toast*`, tombol kembali
  ke Suppliers, `t('setStatusToForCount')` pada modal ubah status massal, tombol
  status `labels.active/inactive/archived`, `labels.updating`, modal hapus
  (`deleteProduct`/`deleteProductDescription`), `displayName` ImportWizard.
- `ProductTable.svelte` — header tabel pakai `labels.productName/category/
  unitOfMeasure/price/stock/status`, aria-label loading/select-all/select-produk
  (`t('selectProductWithName')`), empty state, copy SKU/barcode, `statusLabel`.
- `ProductFormModal.svelte` — judul modal add/edit, error validasi
  (`error*`), label form, status dropdown, pricing rules
  (`t('noPricingRulesBasePrice')`, `t('minWithValue')`), footer
  `labels.saving/add/update/cancel`.
- `ProductDetailDrawer.svelte` — `statusLabel`, `t('weightKg'/'weightGram')`,
  `t('storeWithId')`, `t('classWithId')`, label finansial, `visibleOnlyToAdmins`,
  `copiedToClipboard`, audit trail, footer hapus/edit.
- `ProductFiltersToolbar.svelte` — placeholder search, `statusLabel`, chip
  (`t('categoriesCount'/'supplierWithName')`), `labels.lowStock`, dropdown status,
  tombol Add (`addProductTitle`/`requiresProductCreatePermission`).
- `ProductBulkActions.svelte` — `t('selectedCountLabel')`, `changeStatus`, `clear`.
- `ProductActionsDropdown.svelte` — `actions`, `productActions`, `lihatDetail`,
  `adjustStock`, `editProduk`, `deleteProduct`.
- `CategoryFilterModal.svelte` — `filterKategori`, `filterProduk`, `cariNama`,
  `allCategoriesAZ`, `noCategoriesFound`, `resetAll`, `applyFilter`.

Key i18n baru (mirror id/en), kelompok `// ===== Products =====`:
`loadingProducts`, `tryAdjustingOrAddFirstProduct`, `selectProductWithName`,
`allStatus`, `categoriesCount`, `categoriesSelectedCount`, `supplierWithName`,
`requiresProductCreatePermission`, `addProductTitle`, `selectedCountLabel`,
`productActions`, `copiedToClipboard`, `stokLogistik`, `weightKg`, `weightGram`,
`storeWithId`, `hargaBeliDanMargin`, `hiddenParenthetical`,
`visibleOnlyToAdmins`, `classWithId`, `backToSuppliers`, `supplierWithId`,
`changeStatus`, `setStatusToPrefix`, `setStatusToForCount`, `updating`,
`deleteProductDescription`, `allCategoriesAZ`, `resetAll`, `applyFilter`,
`optionalShort`, `errorNameRequired`, `errorSkuRequired`,
`errorCategoryRequired`, `errorPricePositive`, `errorStockNonNegative`,
`noPricingRulesBasePrice`, `minWithValue`, dan `toast*` (ProductImportCompleted,
AllSelectedAlreadyStatus, UpdatedProductsToStatus, ProductsAlreadyStatus,
FailedToUpdateStatuses, StockAdjusted, FailedToAdjustStock,
FailedToLoadCategories, FailedToLoadBrands, FailedToLoadTaxClasses,
FailedToLoadUnitsOfMeasure, InsufficientPermissionToAdd, ProductAdded,
FailedToAddProduct, InsufficientPermissionToUpdate, NoProductSelected,
ProductUpdated, FailedToUpdateProduct, ProductDeleted, FailedToDeleteProduct).
Nilai `adjustStock` di `id.ts` diubah 'Adjust Stock' → 'Atur Stok' (sebelumnya
belum dipakai komponen; hanya muncul di diksionari).

Test: `npx vitest run src/shared/i18n src/modules/product` → 101/101 hijau.
`svelte-check` penuh → tetap hanya 3 error pre-existing
(EditStorageLocationModal.svelte dan useRBAC.test.ts).

## Catatan Sesi 2026-08-09 (review + perbaikan)

Review agent atas seluruh restrukturisasi (42 file `.svelte`, 533 test) →
**tidak ada isu blocking**; semua key `labels.*`/`t()` terverifikasi ada di
`id.ts`, mirror `en` sesuai, build production sukses. Temuan low-severity yang
diperbaiki:

1. `t()` di `web/src/shared/i18n/index.svelte.ts:66` hanya replace placeholder
   pertama (`replace` tanpa flag `g`) → diganti `replaceAll` (footgun untuk key
   baru yang mengulang placeholder).
2. Default prop `ConfirmDeleteModal` (title/confirmLabel/cancelLabel) ditangkap
   saat mount (`$bindable(labels.confirmDelete)` dst.) → default diubah `''`,
   fallback `labels.*` dievaluasi reaktif di template (`title || labels.confirmDelete`
   dst.), tetap `$bindable` untuk binding eksternal.
3. `web/src/shared/constants/labels.ts` (shim re-export lama) — **tidak ada
   importer** → dihapus; import aktif memakai `$shared/constants/permissions`
   dan `$shared/constants/roles`.

Temuan yang dibiarkan: fallback status-label `'- '` (visual, bukan teks; cabang
`labels.draft/discontinued` sudah terlokalisasi) dan `document.title` non-reaktif
ganti locale (minor, refresh saat navigasi).

Test: i18n 5/5 hijau; `svelte-check` penuh → tetap hanya 3 error pre-existing.

## Perintah Verifikasi

```bash
cd web && npx vitest run src/shared/i18n          # fondasi
cd web && npx vitest run <file-test-yang-diupdate> # per komponen
cd web && npx vitest run                           # penuh
cd web && npm run build                            # build
```
