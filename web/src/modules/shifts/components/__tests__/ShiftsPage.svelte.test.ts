import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ShiftsPage.svelte"),
    "utf-8",
  );
}

describe("ShiftsPage.svelte source-structure guards", () => {
  const src = getSource();

  it("binds CashBreakdown total to the closing balance", () => {
    expect(src).toContain("<CashBreakdown bind:total={closingBalance} />");
  });

  it("enables closing the shift for a zero or positive balance", () => {
    expect(src).toContain("disabled={isSubmitting || closingBalance < 0}");
  });

  it("does not disable closing the shift at zero balance", () => {
    expect(src).not.toContain("closingBalance <= 0");
  });

  it("guards closing against a negative balance in the handler", () => {
    expect(src).toContain("if (closingBalance < 0) return;");
  });
});
