import {
  consignmentPrint,
  type ConsignmentPrintDoc,
} from "../stores/consignmentPrint.svelte";

/**
 * Render a consignment receipt/return document into the print overlay and open
 * the browser print dialog. Mirrors the sale-receipt `previewPrint` path in
 * `$shared/services/print-service.ts` — the document prints, then the overlay is
 * cleared so the next print starts clean.
 */
export function printConsignmentDocument(doc: ConsignmentPrintDoc): void {
  consignmentPrint.set(doc);
  setTimeout(() => {
    window.print();
    setTimeout(() => consignmentPrint.set(null), 1000);
  }, 300);
}
