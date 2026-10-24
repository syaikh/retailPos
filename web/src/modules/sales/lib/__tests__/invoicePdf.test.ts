import { describe, it, expect, vi, beforeEach } from "vitest";

// jsPDF's doc.save() falls back to Node's fs when there is no real browser
// download, which used to dump invoice-INV-*.pdf into the working directory.
// Mock both jspdf modules so the tests never touch the filesystem; the saved
// filename is asserted on the spy instead.
const { saveSpy } = vi.hoisted(() => ({ saveSpy: vi.fn() }));

vi.mock("jspdf", () => {
  class MockJsPDF {
    lastAutoTable = { finalY: 0 };
    setFontSize() {}
    text() {}
    save(filename: string) {
      saveSpy(filename);
    }
  }
  return { jsPDF: MockJsPDF };
});

vi.mock("jspdf-autotable", () => ({
  default: (doc: { lastAutoTable: { finalY: number } }) => {
    doc.lastAutoTable.finalY = 70;
  },
}));

describe("invoicePdf", () => {
  beforeEach(() => {
    saveSpy.mockClear();
  });

  it("returns false for null transaction", async () => {
    const { downloadInvoice } = await import("../invoicePdf");
    const result = await downloadInvoice(null as never, (d: Date) =>
      d.toISOString(),
    );
    expect(result).toBe(false);
    expect(saveSpy).not.toHaveBeenCalled();
  });

  it("exports downloadInvoice function", async () => {
    const mod = await import("../invoicePdf");
    expect(typeof mod.downloadInvoice).toBe("function");
  });

  it("returns true for basic transaction", async () => {
    const { downloadInvoice } = await import("../invoicePdf");
    const result = await downloadInvoice(
      {
        invoice_number: "INV-001",
        created_at: "2025-01-15T10:00:00Z",
        total_amount: 150000,
      },
      (d: Date) => d.toISOString(),
    );
    expect(result).toBe(true);
    expect(saveSpy).toHaveBeenCalledWith("invoice-INV-001.pdf");
  });

  it("returns true with customer name", async () => {
    const { downloadInvoice } = await import("../invoicePdf");
    const result = await downloadInvoice(
      {
        invoice_number: "INV-002",
        created_at: "2025-01-15T10:00:00Z",
        customer_name: "John Doe",
        total_amount: 100000,
      },
      (d: Date) => d.toISOString(),
    );
    expect(result).toBe(true);
    expect(saveSpy).toHaveBeenCalledWith("invoice-INV-002.pdf");
  });

  it("returns true with items", async () => {
    const { downloadInvoice } = await import("../invoicePdf");
    const result = await downloadInvoice(
      {
        invoice_number: "INV-003",
        created_at: "2025-01-15T10:00:00Z",
        payment_method: "cash",
        items: [
          { name: "Product A", quantity: 2, unit_price: 50000 },
          { name: "Product B", quantity: 1, unit_price: 30000 },
        ],
        total_amount: 130000,
      },
      (d: Date) => d.toISOString(),
    );
    expect(result).toBe(true);
    expect(saveSpy).toHaveBeenCalledWith("invoice-INV-003.pdf");
  });

  it("returns true with tax breakdown", async () => {
    const { downloadInvoice } = await import("../invoicePdf");
    const result = await downloadInvoice(
      {
        invoice_number: "INV-004",
        created_at: "2025-01-15T10:00:00Z",
        items: [{ name: "Item X", quantity: 1, unit_price: 100000 }],
        total_amount: 111000,
        tax: 11000,
      },
      (d: Date) => d.toISOString(),
    );
    expect(result).toBe(true);
    expect(saveSpy).toHaveBeenCalledWith("invoice-INV-004.pdf");
  });
});
