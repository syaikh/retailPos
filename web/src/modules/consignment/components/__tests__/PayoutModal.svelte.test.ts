import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "PayoutModal.svelte"),
    "utf-8",
  );
}

describe("PayoutModal.svelte source-structure guards", () => {
  const src = getSource();

  it("imports Modal, Button, SelectSearch from $shared/ui", () => {
    expect(src).toContain("Modal");
    expect(src).toContain("Button");
    expect(src).toContain("SelectSearch");
    expect(src).toContain('$shared/ui');
  });

  it("imports createPayout and listPaymentMethods from consignment-service", () => {
    expect(src).toContain("createPayout");
    expect(src).toContain("listPaymentMethods");
    expect(src).toContain("consignment-service");
  });

  it("exports expected props: settlement, show, onclose, onpaid", () => {
    expect(src).toContain("settlement");
    expect(src).toContain("show");
    expect(src).toContain("onclose");
    expect(src).toContain("onpaid");
  });

  it("imports FormattedNumberInput for the amount field", () => {
    expect(src).toContain("FormattedNumberInput");
  });

  it("imports formatCurrency utility", () => {
    expect(src).toContain("formatCurrency");
    expect(src).toContain("../lib/format");
  });

  it("uses Modal with bind:open and footer snippet", () => {
    expect(src).toContain("bind:open={show}");
    expect(src).toContain("{#snippet footer()}");
  });

  it("has paying state to track submission loading", () => {
    expect(src).toContain("paying");
    expect(src).toContain("$state");
  });

  it("has paymentMethods state populated via listPaymentMethods", () => {
    expect(src).toContain("paymentMethods");
    expect(src).toContain("listPaymentMethods()");
  });

  it("validates payment method and amount before submit", () => {
    expect(src).toContain("payment_method_id");
    expect(src).toContain("form.amount <= 0");
  });

  it("calls createPayout with settlement.id and form payload", () => {
    expect(src).toContain("createPayout(settlement.id");
  });

  it("calls onpaid callback after successful payout", () => {
    expect(src).toContain("onpaid()");
  });

  it("shows saving text while paying is true", () => {
    expect(src).toContain("labels.saving");
  });
});
