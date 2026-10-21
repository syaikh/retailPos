import { describe, it, expect } from "vitest";

// Singleton guard for the session-kernel split: $modules/auth re-exports the
// refresh kernel, session helpers and auth store from $shared. If someone
// ever copies the implementation back into the module (instead of
// re-exporting), these references diverge and the refresh lock / store state
// silently forks into two instances — refresh dedup and cross-tab leadership
// would break. This test fails before that ships.
describe("session kernel identity", () => {
  it("auth-service re-exports the same functions as the $shared kernel", async () => {
    const viaModule = await import("../auth-service");
    const viaShared = await import("$shared/auth/refresh-kernel");

    expect(viaModule.refreshAccessToken).toBe(viaShared.refreshAccessToken);
    expect(viaModule.setupAxiosInterceptors).toBe(
      viaShared.setupAxiosInterceptors,
    );
    expect(viaModule.logout).toBe(viaShared.logout);
    expect(viaModule.startProactiveRefresh).toBe(
      viaShared.startProactiveRefresh,
    );
    expect(viaModule.stopProactiveRefresh).toBe(
      viaShared.stopProactiveRefresh,
    );
    expect(viaModule.markPasswordChangeRequired).toBe(
      viaShared.markPasswordChangeRequired,
    );
    expect(viaModule.isPasswordChangeRequired).toBe(
      viaShared.isPasswordChangeRequired,
    );
    expect(viaModule.PASSWORD_CHANGE_REQUIRED).toBe(
      viaShared.PASSWORD_CHANGE_REQUIRED,
    );
  });

  // 20s timeout: the first import of the auth barrel compiles
  // PasswordChangeForm.svelte and its whole graph, which blows past the 5s
  // default in CI-cold workers. The check itself is a reference comparison.
  it(
    "the auth barrel resolves to the same auth store instance",
    async () => {
      const viaModule = await import("$modules/auth");
      const viaShared = await import("$shared/stores/auth.svelte");

      expect(viaModule.useAuthStore).toBe(viaShared.useAuthStore);
    },
    20000,
  );

  it("the auth module's session shim resolves to the shared helpers", async () => {
    const viaShim = await import("../../lib/session");
    const viaShared = await import("$shared/api/session");

    expect(viaShim.getAuthToken).toBe(viaShared.getAuthToken);
    expect(viaShim.setAccessToken).toBe(viaShared.setAccessToken);
    expect(viaShim.removeAccessToken).toBe(viaShared.removeAccessToken);
  });
});
