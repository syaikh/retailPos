import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ReturnPage.svelte"),
    "utf-8",
  );
}

describe("ReturnPage.svelte source-structure guards", () => {
  const src = getSource();

  it("uses HTMLSelectElement for pending return select onchange cast", () => {
    expect(src).toContain("(e.target as HTMLSelectElement).value");
  });

  it("does not use HTMLInputElement for the pending return select", () => {
    const pendingReturnBlock = src.slice(src.indexOf("pending_return_id"));
    expect(pendingReturnBlock).not.toContain(
      "(e.target as HTMLInputElement).value",
    );
  });

  it("clears product_id when pending return is unlinked", () => {
    expect(src).toContain("line.product_id = undefined");
  });

  it("resets reason to 'other' when pending return is unlinked", () => {
    expect(src).toContain('line.reason = "other"');
  });

  it("has else branch that resets fields after pending return selection check", () => {
    const ifBlock = src.indexOf("if (prId)");
    expect(ifBlock).toBeGreaterThan(-1);
    const elseBlock = src.indexOf("} else {", ifBlock);
    expect(elseBlock).toBeGreaterThan(-1);
    const afterElse = src.slice(elseBlock, elseBlock + 200);
    expect(afterElse).toContain("line.product_id = undefined");
    expect(afterElse).toContain('line.reason = "other"');
  });
});
