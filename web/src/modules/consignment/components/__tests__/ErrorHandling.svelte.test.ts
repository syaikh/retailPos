import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(component: string): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", component),
    "utf-8",
  );
}

const components = [
  "ArrangementsPage.svelte",
  "PendingReturnPage.svelte",
  "ReceiptEntry.svelte",
  "ReturnPage.svelte",
  "SettlementPage.svelte",
  "TermsEditor.svelte",
];

describe("Consignment components error-handling pattern", () => {
  for (const component of components) {
    it(`${component} uses getApiErrorMessage`, () => {
      const src = getSource(component);
      expect(src).toContain("getApiErrorMessage");
    });

    it(`${component} does not use old e instanceof Error pattern`, () => {
      const src = getSource(component);
      expect(src).not.toContain("e instanceof Error");
    });
  }
});
