import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "Dropdown.svelte"),
    "utf-8",
  );
}

describe("Dropdown.svelte source-structure guards", () => {
  const src = getSource();

  it("coalesces scroll/resize repositioning into one frame", () => {
    // The reposition listeners are window-level capture handlers, so an
    // unthrottled handler forced a getBoundingClientRect plus a $state write
    // per scroll event per open menu (dropdowns are per table row).
    expect(src).toContain("let positionRafId = 0;");
    expect(src).toContain("if (positionRafId) return;");
    expect(src).toContain("positionRafId = requestAnimationFrame(() => {");
    expect(src).toContain("positionRafId = 0;\n        computePosition();");
  });

  it("cancels the pending reposition frame on teardown", () => {
    expect(src).toContain("cancelAnimationFrame(positionRafId);");
  });

  it("keeps the single reposition definition per open effect", () => {
    expect(src.split("function reposition()").length - 1).toBe(1);
    expect(src.split("let positionRafId = 0;").length - 1).toBe(1);
    expect(src).toContain('window.addEventListener("scroll", reposition');
    expect(src).toContain('window.addEventListener("resize", reposition');
  });
});
