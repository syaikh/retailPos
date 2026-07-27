<script lang="ts">
  import { printReceipt } from '$shared/stores/printReceipt.svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
</script>

{#if $printReceipt}
<div class="thermal-receipt-container">
  <div class="thermal-receipt" id="thermal-receipt">
    <div class="thermal-shop-name">RETAIL POS</div>
    <div class="thermal-row">
      <span class="thermal-label">Invoice:</span>
      <span class="thermal-value">{$printReceipt.invoice_number}</span>
    </div>
    <div class="thermal-row">
      <span class="thermal-label">Waktu:</span>
      <span class="thermal-value">{formatDateTimeInJakarta($printReceipt.created_at || new Date().toISOString())}</span>
    </div>
    {#if $printReceipt.customer_name}
    <div class="thermal-row">
      <span class="thermal-label">Customer:</span>
      <span class="thermal-value">{$printReceipt.customer_name}</span>
    </div>
    {/if}
    <div class="thermal-divider"></div>
    {#each $printReceipt.items as item}
      <div class="thermal-item">
        <div class="thermal-item-name">{item.name} x{item.quantity}</div>
        <div class="thermal-item-price">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</div>
      </div>
    {/each}
    <div class="thermal-divider"></div>
    {#if $printReceipt.subtotal_dpp != null && $printReceipt.tax != null && $printReceipt.tax > 0}
      <div class="thermal-row">
        <span class="thermal-label">DPP</span>
        <span class="thermal-value">{$printReceipt.subtotal_dpp.toLocaleString('id-ID')}</span>
      </div>
      <div class="thermal-row">
        <span class="thermal-label">PPN 11%</span>
        <span class="thermal-value">{$printReceipt.tax.toLocaleString('id-ID')}</span>
      </div>
      <div class="thermal-divider-thin"></div>
    {/if}
    <div class="thermal-item thermal-item-total">
      <span>TOTAL</span>
      <span>{$printReceipt.total_amount.toLocaleString('id-ID')}</span>
    </div>
    <div class="thermal-payment-section">
      <div class="thermal-row">
        <span class="thermal-label">Pembayaran</span>
        <span></span>
      </div>
      {#if $printReceipt.payments && $printReceipt.payments.length > 0}
        {#each $printReceipt.payments as p}
          <div class="thermal-payment-row">
            <span class="thermal-payment-method">{p.method === 'CASH' ? 'Tunai' : p.method}</span>
            <span class="thermal-payment-amount">{p.amount.toLocaleString('id-ID')}</span>
          </div>
        {/each}
      {:else}
        <div class="thermal-payment-row">
          <span class="thermal-payment-value">{$printReceipt.paymentMethod}</span>
        </div>
      {/if}
    </div>
    <div class="thermal-divider"></div>
    <div class="thermal-footer">
      <p>Terima kasih atas kunjungan Anda!</p>
      <p>Barang yang sudah dibeli tidak dapat dikembalikan.</p>
    </div>
  </div>
</div>
{/if}
