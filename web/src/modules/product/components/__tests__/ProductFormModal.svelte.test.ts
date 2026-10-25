import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ProductFormModal.svelte"),
    "utf-8",
  );
}

describe("ProductFormModal.svelte source-structure guards", () => {
  const src = getSource();
  const count = (needle: string) => src.split(needle).length - 1;

  it("coalesces the category and brand dropdown repositioning", () => {
    // Both dropdowns listen for window scroll/resize with capture; the
    // unthrottled handlers did one forced layout read + reactive write per
    // scroll event for each open dropdown.
    expect(count("let positionRafId = 0;")).toBe(2);
    expect(count("function reposition()")).toBe(2);
    expect(count("if (positionRafId) return;")).toBe(2);
    expect(count("positionRafId = requestAnimationFrame(() => {")).toBe(2);
    expect(count("cancelAnimationFrame(positionRafId);")).toBe(2);
  });

  it("repositions each dropdown inside its scheduled frame", () => {
    expect(src).toContain(
      "positionRafId = 0;\n        computeCategoryPosition();",
    );
    expect(src).toContain(
      "positionRafId = 0;\n        computeBrandPosition();",
    );
  });

  it("removes both reposition listeners on teardown", () => {
    const scrollOff = 'window.removeEventListener("scroll", reposition';
    const resizeOff = 'window.removeEventListener("resize", reposition)';
    expect(count(scrollOff)).toBe(2);
    expect(count(resizeOff)).toBe(2);
    expect(count('window.addEventListener("scroll", reposition')).toBe(2);
    expect(count('window.addEventListener("resize", reposition')).toBe(2);
  });
});
