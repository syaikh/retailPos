import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "main.svelte"),
    "utf-8",
  );
}

describe("main.svelte source-structure guards", () => {
  const src = getSource();

  it("imports the blocking force-change modal", () => {
    expect(src).toContain(
      'import ForceChangePasswordModal from "$modules/auth/components/ForceChangePasswordModal.svelte"',
    );
    expect(src).toContain("const authStore = useAuthStore()");
  });

  it("blocks rendering while a first-login password rotation is owed", () => {
    const gate = src.indexOf("{:else if authStore.mustChangePassword}");
    const login = src.indexOf('{:else if currentPath === "/login"}');
    expect(gate).toBeGreaterThan(-1);
    expect(login).toBeGreaterThan(gate);
    expect(src).toContain("<ForceChangePasswordModal />");
  });
});
