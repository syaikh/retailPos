import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "StoreOnboardingWizard.svelte"),
    "utf-8",
  );
}

describe("StoreOnboardingWizard.svelte source-structure guards", () => {
  const src = getSource();

  it("wires the module dependencies through their public exports", () => {
    expect(src).toContain(
      'import { getRoles, createUser } from "$modules/admin"',
    );
    expect(src).toContain(
      'import { createStorageLocation } from "$modules/storage-location"',
    );
    expect(src).toContain(
      'import { createStoreAndGet, getReadiness } from "../services/stores-service"',
    );
  });

  it("opens as a bindable modal and resets state on open", () => {
    expect(src).toContain("open = $bindable(false)");
    expect(src).toContain("<Modal\n  bind:open");
    expect(src).toContain("$effect(() => {\n    if (open) {");
    expect(src).toContain('step = "details"');
    expect(src).toContain("createdStore = null");
    expect(src).toContain("void loadRoles()");
  });

  it("defines the five onboarding steps in order", () => {
    expect(src).toContain(
      'const steps: Step[] = ["details", "staff", "location", "stock", "readiness"];',
    );
    expect(src).toContain('case "readiness":');
    expect(src).toContain("return labels.onboardingStepReadiness;");
  });

  it("requires name, address and phone before creating the store", () => {
    expect(src).toContain("const canSubmitDetails = $derived(");
    expect(src).toContain("storeForm.name.trim().length > 0 &&");
    expect(src).toContain("storeForm.address.trim().length > 0 &&");
    expect(src).toContain("storeForm.phone.trim().length > 0 &&");
    expect(src).toContain("disabled={!canSubmitDetails}");
    expect(src).toContain("async function handleDetails()");
    expect(src).toContain("const created = await createStoreAndGet({");
  });

  it("creates staff with the store id and the forced password rotation flag", () => {
    expect(src).toContain("async function handleStaff()");
    expect(src).toContain("await createUser({");
    expect(src).toContain("store_id: createdStore.id,");
    expect(src).toContain("must_change_password: true,");
    expect(src).toContain("role_id: roleRecord.id,");
    expect(src).toContain(
      'errorMsg = t("onboardingStaffFailed", { count: failed })',
    );
    expect(src).toContain("row.error = labels.roleNotFound;");
  });

  it("generates a random temporary password per enabled row", () => {
    expect(src).toContain("function generatePassword(): string {");
    expect(src).toContain("crypto.getRandomValues(new Uint8Array(16))");
    expect(src).toContain("function regeneratePassword(row: StaffRow)");
    expect(src).toContain("labels.regeneratePassword");
  });

  it("pre-fills and validates the default storage location", () => {
    expect(src).toContain("async function handleLocation()");
    expect(src).toContain("code: `LOC-${created.id}`.toUpperCase()");
    expect(src).toContain("name: labels.defaultStorageLocation");
    expect(src).toContain("errorMsg = labels.storageLocationRequired;");
    expect(src).toContain("await createStorageLocation({");
    expect(src).toContain("store_id: createdStore.id,");
  });

  it("offers stock intake as deep links only", () => {
    expect(src).toContain('onclick={() => gotoStock("/inventory/products")}');
    expect(src).toContain('onclick={() => gotoStock("/purchase-orders")}');
    expect(src).toContain(
      'onclick={() => gotoStock("/stock-opnames/adjustments")}',
    );
    expect(src).toContain("function gotoStock(path: string)");
  });

  it("loads readiness when the final step is reached and can refresh it", () => {
    expect(src).toContain(
      '$effect(() => {\n    if (step === "readiness" && createdStore) {',
    );
    expect(src).toContain("readiness = await getReadiness(createdStore.id);");
    expect(src).toContain("async function refreshReadiness()");
  });

  it("mirrors the backend catalog blocker rule in the checklist", () => {
    // Store.Readiness blocks on active==0 OR zeroStock==active; a single
    // >0 check would light the checklist green on an entirely empty catalog.
    expect(src).toContain(
      "{#if readiness.catalog.active_products > 0 && readiness.catalog.zero_stock_products < readiness.catalog.active_products}",
    );
  });

  it("guards replayed steps against duplicate side effects", () => {
    expect(src).toContain(
      '    if (createdStore) {\n      step = "staff";\n      return;\n    }',
    );
    expect(src).toContain("let locationCreated = $state(false)");
    expect(src).toContain(
      '    if (locationCreated) {\n      step = "stock";\n      return;\n    }',
    );
  });

  it("maps every backend blocker code to a localized label", () => {
    expect(src).toContain("function blockerLabel(code: string): string {");
    expect(src).toContain(
      'if (code === "store.address") return labels.blockerAddress;',
    );
    expect(src).toContain(
      'if (code === "store.phone") return labels.blockerPhone;',
    );
    expect(src).toContain(
      'if (code === "store.inactive") return labels.blockerInactive;',
    );
    expect(src).toContain(
      'if (code === "storage_location") return labels.blockerStorageLocation;',
    );
    expect(src).toContain(
      'if (code === "catalog") return labels.blockerCatalog;',
    );
    expect(src).toContain('code.startsWith("staff.")');
    expect(src).toContain(
      't("blockerStaffRole", { role: roleLabel(code.slice(6)) })',
    );
  });

  it("offers follow-up navigation and a finish action", () => {
    expect(src).toContain("onclick={() => gotoStock(");
    expect(src).toContain("labels.goToPos");
    expect(src).toContain("labels.goToUsers");
    expect(src).toContain("function finish()");
    expect(src).toContain("toast.success(labels.onboardingFinished)");
    expect(src).toContain("onComplete()");
  });

  it("localizes the step indicator", () => {
    expect(src).toContain(
      't("stepOf", { current: stepIndex + 1, total: steps.length })',
    );
    expect(src).toContain("labels.onboardingWizardTitle");
    expect(src).toContain('role="alert"');
  });
});
