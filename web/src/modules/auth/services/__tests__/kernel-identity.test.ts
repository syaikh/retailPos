import { describe, it, expect } from "vitest";

// Session kernel identity: $modules/auth re-exports the refresh kernel from
// $shared/auth/refresh-kernel. If someone ever copies the implementation back
// into the module (instead of re-exporting), these references diverge and the
// refresh lock silently forks into two instances — refresh dedup and
// cross-tab leadership would break. This test fails before that ships.
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
    expect(viaModule.stopProactiveRefresh).toBe(viaShared.stopProactiveRefresh);
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
  // default in CI-cold workers.
  // Session state (store + token helpers) is $shared API since PR2 of
  // docs/design/frontend-import-boundary-enforcement-plan.md: the auth module
  // owns flows only, and the shims at modules/auth/lib/session.ts and
  // modules/auth/stores/auth-store.svelte.ts are deleted. Re-adding a
  // session-state export here would silently fork the store into two
  // instances again — consumers must import from $shared directly.
  it("the auth barrel does not export session state", async () => {
    const viaModule = (await import("$modules/auth")) as Record<
      string,
      unknown
    >;

    expect(viaModule.useAuthStore).toBeUndefined();
    expect(viaModule.getAuthToken).toBeUndefined();
  }, 20000);
});
