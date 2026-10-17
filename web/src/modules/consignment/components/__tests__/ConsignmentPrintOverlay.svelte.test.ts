import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ConsignmentPrintOverlay.svelte"),
    "utf-8",
  );
}

describe("ConsignmentPrintOverlay.svelte source-structure guards", () => {
  const src = getSource();

  it("imports the consignment print store and settings branding", () => {
    expect(src).toContain(
      'import { consignmentPrint } from "../stores/consignmentPrint.svelte"',
    );
    expect(src).toContain(
      'import { settingsStore } from "$shared/stores/settings.svelte"',
    );
  });

  it("renders only when a document is queued and uses the print container", () => {
    expect(src).toContain("{#if current}");
    expect(src).toContain('class="consignment-print-container"');
  });

  it("branches on document kind for the title and number label", () => {
    expect(src).toContain('current.kind === "receipt"');
    expect(src).toContain("labels.consignmentReceiptDocumentTitle");
    expect(src).toContain("labels.consignmentReturnDocumentTitle");
    expect(src).toContain("labels.consignmentReceiptNo");
    expect(src).toContain("labels.consignmentReturnNo");
  });

  it("renders a receipt table with accepted qty, price and store share", () => {
    expect(src).toContain("item.accepted_qty");
    expect(src).toContain("item.accepted_qty * item.price");
    expect(src).toContain("item.store_share_type");
  });

  it("renders a return table with qty and reason", () => {
    expect(src).toContain("RETURN_REASON_LABELS[item.reason]");
    expect(src).toContain("item.qty");
  });

  it("renders store and supplier signature blocks", () => {
    expect(src).toContain("consignment-print-signatures");
    expect(src).toContain("labels.consignmentSignatureStore");
    expect(src).toContain("labels.consignmentSignatureSupplier");
  });
});
