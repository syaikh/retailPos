<script lang="ts">
  import { consignmentPrint } from "../stores/consignmentPrint.svelte";
  import { settingsStore } from "$shared/stores/settings.svelte";
  import { labels } from "$shared/i18n";
  import { formatCurrency, formatDateTime } from "../lib/format";
  import { RETURN_REASON_LABELS } from "../types";

  const current = $derived($consignmentPrint);

  const receiptTotal = $derived(
    current?.kind === "receipt"
      ? (current.doc.items || []).reduce(
          (sum, item) => sum + item.accepted_qty * item.price,
          0,
        )
      : 0,
  );
</script>

{#if current}
  <div class="consignment-print-container">
    <div class="consignment-print-doc">
      <header class="consignment-print-header">
        {#if settingsStore.logoPath}
          <img
            class="consignment-print-logo"
            src={settingsStore.logoPath}
            alt=""
          />
        {/if}
        <div class="consignment-print-store">{settingsStore.storeName}</div>
        {#if settingsStore.storeAddress}
          <div class="consignment-print-store-detail">
            {settingsStore.storeAddress}
          </div>
        {/if}
        {#if settingsStore.storePhone}
          <div class="consignment-print-store-detail">
            {labels.phone}: {settingsStore.storePhone}
          </div>
        {/if}
      </header>

      <h1 class="consignment-print-title">
        {current.kind === "receipt"
          ? labels.consignmentReceiptDocumentTitle
          : labels.consignmentReturnDocumentTitle}
      </h1>

      <div class="consignment-print-meta">
        <div>
          <span
            >{current.kind === "receipt"
              ? labels.consignmentReceiptNo
              : labels.consignmentReturnNo}:</span
          >
          <strong
            >{current.kind === "receipt"
              ? current.doc.receipt_number
              : current.doc.return_number}</strong
          >
        </div>
        <div>
          <span>{labels.consignmentDate}:</span>
          {formatDateTime(
            current.kind === "receipt"
              ? current.doc.received_at
              : current.doc.returned_at,
          )}
        </div>
        <div>
          <span>{labels.consignmentSupplier}:</span>
          {current.doc.supplier_name || "-"}
        </div>
        <div>
          <span>{labels.consignmentDocumentArrangement}:</span>
          #{current.doc.arrangement_id}
        </div>
        <div>
          <span
            >{current.kind === "receipt"
              ? labels.consignmentReceivedBy
              : labels.consignmentReturnedBy}:</span
          >
          {current.kind === "receipt"
            ? current.doc.received_by_username || "-"
            : current.doc.returned_by_username || "-"}
        </div>
      </div>

      {#if current.kind === "receipt"}
        <table class="consignment-print-table">
          <thead>
            <tr>
              <th>{labels.consignmentProduct}</th>
              <th class="text-right">{labels.consignmentAccepted}</th>
              <th class="text-right">{labels.consignmentPricePerUnit}</th>
              <th class="text-right">{labels.consignmentTotalValue}</th>
              <th class="text-right">{labels.consignmentStoreShare}</th>
            </tr>
          </thead>
          <tbody>
            {#each current.doc.items || [] as item, i (i)}
              <tr>
                <td>
                  {item.product_name || `#${item.product_id}`}
                  {#if item.product_sku}
                    <br /><small>{item.product_sku}</small>
                  {/if}
                </td>
                <td class="text-right">{item.accepted_qty}</td>
                <td class="text-right">{formatCurrency(item.price)}</td>
                <td class="text-right">
                  {formatCurrency(item.accepted_qty * item.price)}
                </td>
                <td class="text-right">
                  {item.store_share_type === "percentage"
                    ? `${item.store_share_value}%`
                    : formatCurrency(item.store_share_value)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>

        <div class="consignment-print-total">
          {labels.consignmentTotalValue}: {formatCurrency(receiptTotal)}
        </div>
      {:else}
        <table class="consignment-print-table">
          <thead>
            <tr>
              <th>{labels.consignmentProduct}</th>
              <th class="text-right">{labels.consignmentQty}</th>
              <th>{labels.consignmentReason}</th>
            </tr>
          </thead>
          <tbody>
            {#each current.doc.items || [] as item, i (i)}
              <tr>
                <td>
                  {item.product_name || `#${item.product_id}`}
                  {#if item.product_sku}
                    <br /><small>{item.product_sku}</small>
                  {/if}
                </td>
                <td class="text-right">{item.qty}</td>
                <td>
                  {labels[RETURN_REASON_LABELS[item.reason]] || item.reason}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}

      {#if current.doc.notes}
        <div class="consignment-print-notes">
          <strong>{labels.notes}:</strong>
          {current.doc.notes}
        </div>
      {/if}

      <div class="consignment-print-signatures">
        <div>
          <div class="consignment-print-signature-line">
            {labels.consignmentSignatureStore}
          </div>
        </div>
        <div>
          <div class="consignment-print-signature-line">
            {labels.consignmentSignatureSupplier}
          </div>
        </div>
      </div>

      <div class="consignment-print-footer">
        {labels.consignmentPrintedAt}: {formatDateTime(
          new Date().toISOString(),
        )}
      </div>
    </div>
  </div>
{/if}
