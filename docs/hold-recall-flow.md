# Hold & Recall Transaction Flow

## Status Lifecycle

```
Keranjang → Hold (F6)        → "parked"    (tersimpan, stok TIDAK dikurangi)
         → Recall (F5)       → "recalled"  (dimuat ke cart, status berubah)
         → Checkout (F4+Enter) → recalled sale → "cancelled"
                                 new sale       → "completed" (stok dikurangi)
         → Cancel (tombol)   → "cancelled"
         → Hold lagi setelah recall
             → recalled sale → "cancelled"
             → new sale      → "parked" (re-park)
```

## API Endpoints

| Action | Method | Endpoint | Body penting |
|--------|--------|----------|-------------|
| Hold | POST | `/sales/parked` | `{ items, recalled_sale_id? }` |
| List | GET | `/sales/parked` | — |
| Recall | POST | `/sales/parked/:id/recall` | — |
| Cancel | DELETE | `/sales/parked/:id` | — |
| Checkout | POST | `/sales` | `{ items, ..., parked_sale_id? }` |

## Detail Alur

### 1. Hold (F6)

Frontend (`PosPage.svelte`):
- Guard: cart tidak boleh kosong
- Items dikirim minimal: `{ product_id, quantity, subtotal }`
- Juga mengirim `payment_method` dan `recalled_sale_id` jika ada (re-park)

Backend (`internal/sale/service.go`):
- Jika `recalledSaleID` tidak nil (re-park):
  ```sql
  UPDATE sales SET status = 'cancelled' WHERE id = $1 AND status = 'recalled'
  ```
- Insert sale baru dengan `status = 'parked'`
- **Tidak ada pengecekan atau pengurangan stok**
- Cart dibersihkan, counter recall di-refresh

### 2. Recall (F5)

Frontend:
- `fetchParkedSales()` → `GET /sales/parked`
- Tampilkan daftar di `ParkedSalesModal`
- Klik Recall → `POST /sales/parked/:id/recall`

Backend:
```sql
UPDATE sales SET status = 'recalled', updated_at = NOW()
WHERE id = $1 AND status IN ('parked', 'recalled')
RETURNING id, invoice_number, ..., status, ...
```
- **Idempotent**: bisa recall sale yang sudah `'recalled'`
- Response berisi items → dimuat ke cart

Frontend setelah sukses:
- `recalledSaleId = saleId` (disimpan di state)
- Items dipetakan ke cart array: `id → product_id`, `price → unit_price`
- `resolveCartPrices()` untuk re-resolve pricing rules
- Modal ditutup, daftar parkir di-refresh

### 3. Checkout setelah Recall (F4 → Enter)

Frontend (`finalizeSale()`):
1. **Nilai `recalledSaleId` ditangkap** sebelum modal checkout ditutup
2. `processCheckout(capturedRecalledSaleId)` dipanggil
3. Request `POST /sales` dengan `parked_sale_id`

Backend (`CreateSaleWithParkedSale()`):
1. `SELECT status FROM sales WHERE id = $1 AND status = 'recalled' FOR UPDATE`
   - Lock row, verifikasi status masih `'recalled'`
   - Jika tidak ditemukan → HTTP 409 `"parked sale already checked out or cancelled"`
2. Batch cek stok dengan `FOR UPDATE` lock
3. Kurangi stok di `product_stock`
4. Insert sale baru `status = 'completed'`
5. `UPDATE sales SET status = 'cancelled' WHERE id = $1 AND status = 'recalled'`
   — consumed
6. Commit transaksi, publish `sale.created` event

### 4. Direct Checkout (tanpa recall)

- `processCheckout()` tanpa argument (`parkedSaleId = null`)
- Request body tanpa `parked_sale_id`
- Backend panggil `CreateSale()` — cek stok, kurangi stok, insert sale baru

### 5. Cancel

- `DELETE /sales/parked/:id`
- ```sql
  UPDATE sales SET status = 'cancelled' WHERE id = $1 AND status IN ('parked', 'recalled')
  ```
- 204 No Content jika sukses
- 404 jika sale sudah cancelled/completed (tidak ditemukan)

## Poin Penting

- **Stok hanya dikurangi saat checkout final**, tidak saat hold
- **Recall bersifat idempotent** — bisa recall ulang sale yang sudah `'recalled'`
- **Race condition** dicegah dengan `FOR UPDATE` lock pada `CreateSaleWithParkedSale()`
- **Counter recall** dihitung dari sale dengan `status === 'parked'` (sale `'recalled'` tidak dihitung)
- **Repark**: hold saat sedang dalam sesi recall akan meng-cancel sale recalled dan membuat sale parked baru
