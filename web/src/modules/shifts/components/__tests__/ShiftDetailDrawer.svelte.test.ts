import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ShiftDetailDrawer.svelte"),
    "utf-8",
  );
}

describe("ShiftDetailDrawer.svelte source-structure guards", () => {
  const src = getSource();

  it("imports listCashMovements service and CashMovement type", () => {
    expect(src).toContain(
      'import { listCashMovements } from "../services/shift-service"',
    );
    expect(src).toContain('import type { CashMovement } from "../types"');
  });

  it("tracks cash movements in component state", () => {
    expect(src).toContain("let cashMovements = $state<CashMovement[]>([]);");
    expect(src).toContain("let movementsLoading = $state(false);");
  });

  it("loads cash movements when the drawer opens", () => {
    expect(src).toContain("$effect(() => {");
    expect(src).toContain("if (showDetailDrawer && selectedShift) {");
    expect(src).toContain("loadCashMovements(selectedShift.id);");
  });

  it("resets state on load and falls back to empty on failure", () => {
    expect(src).toContain("async function loadCashMovements(shiftId: number) {");
    expect(src).toContain("cashMovements = [];");
    expect(src).toContain("cashMovements = await listCashMovements(shiftId);");
    expect(src).toContain("} catch {");
  });

  it("maps movement types to localized labels", () => {
    expect(src).toContain('function movementLabel(type: CashMovement["type"]) {');
    expect(src).toContain('return type === "paid_in"');
    expect(src).toContain(': type === "paid_out"');
    expect(src).toContain(": labels.cashDrop;");
  });

  it("treats paid_in as the only positive movement", () => {
    expect(src).toContain('function isPositive(type: CashMovement["type"]) {');
    expect(src).toContain('return type === "paid_in";');
  });

  it("renders a cash movements section with loading and empty states", () => {
    expect(src).toContain("{labels.cashMovements}");
    expect(src).toContain("{#if movementsLoading}");
    expect(src).toContain("{labels.loading}");
    expect(src).toContain("{:else if cashMovements.length === 0}");
    expect(src).toContain("{labels.noCashMovements}");
    expect(src).toContain("{#each cashMovements as m (m.id)}");
  });

  it("renders each movement with label, time, description and signed amount", () => {
    expect(src).toContain("{movementLabel(m.type)}");
    expect(src).toContain("{formatDateTime(m.created_at)}");
    expect(src).toContain("{#if m.description}");
    expect(src).toContain("{isPositive(m.type) ? \"+\" : \"-\"}{formatMoney(m.amount)}");
  });
});