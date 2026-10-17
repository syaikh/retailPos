import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { get } from "svelte/store";
import { consignmentPrint } from "../stores/consignmentPrint.svelte";
import { printConsignmentDocument } from "./print";
import type { Receipt } from "../types";

const receipt = {
  id: 1,
  receipt_number: "CR-2026-000001",
  supplier_id: 1,
  arrangement_id: 1,
  store_id: 1,
  received_by: 1,
  received_at: "2026-06-15T00:00:00Z",
  created_at: "2026-06-15T00:00:00Z",
  items: [],
} as Receipt;

describe("printConsignmentDocument", () => {
  const printSpy = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    printSpy.mockClear();
    window.print = printSpy;
  });

  afterEach(() => {
    vi.useRealTimers();
    consignmentPrint.set(null);
  });

  it("queues the document, opens the print dialog, then clears the queue", () => {
    printConsignmentDocument({ kind: "receipt", doc: receipt });

    expect(get(consignmentPrint)).toEqual({ kind: "receipt", doc: receipt });
    expect(printSpy).not.toHaveBeenCalled();

    vi.advanceTimersByTime(300);
    expect(printSpy).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(1000);
    expect(get(consignmentPrint)).toBeNull();
  });
});
