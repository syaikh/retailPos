import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "TermsEditor.svelte"),
    "utf-8",
  );
}

describe("TermsEditor.svelte source-structure guards", () => {
  const src = getSource();

  it("uses HTMLSelectElement for store_share_type select oninput cast", () => {
    expect(src).toContain(
      '"store_share_type",\n                          (e.target as HTMLSelectElement).value,',
    );
  });

  it("does not use HTMLInputElement for the store_share_type select", () => {
    expect(src).not.toContain(
      '"store_share_type",\n                          (e.target as HTMLInputElement).value,',
    );
  });

  it("uses hardcoded min='1' for share value NumberInput (no redundant ternary)", () => {
    const minIdx = src.indexOf('min="1"');
    expect(minIdx).toBeGreaterThan(-1);
    const surrounding = src.slice(minIdx - 200, minIdx + 200);
    expect(surrounding).toContain("store_share_value");
    expect(surrounding).not.toMatch(
      /min=\{.*SHARE_TYPE_PERCENTAGE.*\? "1" : "1"\}/,
    );
  });
});
