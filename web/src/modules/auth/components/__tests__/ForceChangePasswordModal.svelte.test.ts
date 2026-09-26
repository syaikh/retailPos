import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(
      path.dirname(__filename),
      "..",
      "ForceChangePasswordModal.svelte",
    ),
    "utf-8",
  );
}

describe("ForceChangePasswordModal.svelte source-structure guards", () => {
  const src = getSource();

  it("imports changePassword, logout and the auth store from the auth module", () => {
    expect(src).toContain(
      'import { changePassword, logout, useAuthStore } from "$modules/auth"',
    );
    expect(src).toContain("labels.forceChangePasswordTitle");
    expect(src).toContain("labels.forceChangePasswordDesc");
  });

  it("renders its own blocking overlay instead of the shared Modal", () => {
    expect(src).not.toContain("import { Modal");
    expect(src).toContain("fixed inset-0");
    expect(src).toContain("z-[100]");
    expect(src).toContain('role="dialog"');
    expect(src).toContain('aria-modal="true"');
    expect(src).toContain('aria-labelledby="force-change-password-title"');
    expect(src).toContain("{#if auth.mustChangePassword}");
  });

  it("collects current, new and confirmation passwords with labels", () => {
    expect(src).toContain('let currentPassword = $state("")');
    expect(src).toContain('let newPassword = $state("")');
    expect(src).toContain('let confirmPassword = $state("")');
    expect(src).toContain("labels.currentPassword");
    expect(src).toContain("labels.newPassword");
    expect(src).toContain("labels.confirmNewPassword");
    expect(src).toContain('for="force-current-password"');
    expect(src).toContain('for="force-new-password"');
    expect(src).toContain('for="force-confirm-password"');
  });

  it("validates length and confirmation before enabling submit", () => {
    expect(src).toContain("newPassword.length >= 8");
    expect(src).toContain("newPassword === confirmPassword");
    expect(src).toContain("disabled={!canSubmit}");
    expect(src).toContain("tooShort ? labels.passwordMinLength");
    expect(src).toContain("mismatch ? labels.passwordsDoNotMatch");
  });

  it("submits through changePassword and reports the outcome", () => {
    expect(src).toContain("await changePassword(currentPassword, newPassword)");
    expect(src).toContain("toast.success(labels.passwordChangedSuccessfully)");
    expect(src).toContain("errorMsg = result.message;");
    expect(src).toContain('role="alert"');
  });

  it("focuses the first field while the rotation flag is set", () => {
    expect(src).toContain("$effect(() => {");
    expect(src).toContain("if (auth.mustChangePassword)");
    expect(src).toContain("fieldRef?.focus()");
  });

  it("offers an explicit sign-out escape hatch", () => {
    expect(src).toContain("labels.signOut");
    expect(src).toContain("async function handleLogout()");
    expect(src).toContain("await logout()");
  });
});
