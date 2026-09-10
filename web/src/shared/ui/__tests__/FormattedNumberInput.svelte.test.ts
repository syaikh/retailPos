import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "FormattedNumberInput.svelte"),
    "utf-8",
  );
}

describe("FormattedNumberInput.svelte source-structure guards", () => {
  const src = getSource();

  it("uses inputmode numeric for mobile keyboard", () => {
    expect(src).toContain('inputmode="numeric"');
  });

  it("formats value with id-ID locale", () => {
    expect(src).toContain('toLocaleString("id-ID")');
  });

  it("strips non-numeric characters on input", () => {
    expect(src).toContain("/[^0-9]/g");
  });

  it("uses type text to avoid browser numeric spinners", () => {
    expect(src).toContain('type="text"');
  });

  it("has onfocus handler", () => {
    expect(src).toContain("onfocus={handleFocus}");
  });

  it("has onblur handler", () => {
    expect(src).toContain("onblur={handleBlur}");
  });

  it("has oninput handler", () => {
    expect(src).toContain("oninput={handleInput}");
  });

  it("is disabled when disabled prop is true", () => {
    expect(src).toContain("{disabled}");
  });

  it("supports id prop", () => {
    expect(src).toContain("id={id");
  });

  it("supports placeholder prop", () => {
    expect(src).toContain("{placeholder}");
  });
});
