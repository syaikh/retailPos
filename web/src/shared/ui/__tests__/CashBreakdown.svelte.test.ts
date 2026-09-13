import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "CashBreakdown.svelte"),
    "utf-8",
  );
}

describe("CashBreakdown.svelte source-structure guards", () => {
  const src = getSource();

  it("exposes a bindable total prop", () => {
    expect(src).toContain("total = $bindable(0)");
    expect(src).toContain("total?: number");
  });

  it("recomputes total from denomination counts", () => {
    expect(src).toContain("const counts: Record<number, number> = $state({});");
    expect(src).toContain("$effect(() => {");
    expect(src).toContain(
      "(sum, [denom, count]) => sum + Number(denom) * count",
    );
  });

  it("keeps the bound total when no denominations have been entered", () => {
    expect(src).toContain("const entries = Object.entries(counts);");
    expect(src).toContain("if (entries.length === 0) return;");
  });

  it("lists all Rupiah denominations", () => {
    expect(src).toContain('{ label: "100rb", value: 100000 }');
    expect(src).toContain('{ label: "500", value: 500 }');
    expect(src).toContain('{ label: "100", value: 100 }');
  });
});
