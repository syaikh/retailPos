import { describe, it, expect, vi, beforeEach } from "vitest";
import type { AxiosInstance } from "axios";

const mockPost = vi.fn();
vi.mock("axios", () => ({
  default: {
    create: () => ({
      post: mockPost,
      interceptors: {
        response: { use: vi.fn(), eject: vi.fn() },
        request: { use: vi.fn(), eject: vi.fn() },
      },
    }),
    isAxiosError: (err: unknown) =>
      typeof err === "object" && err !== null && "isAxiosError" in err,
  },
}));

vi.mock("$shared/utils/tab-coordination", () => ({
  initTabCoordination: vi.fn(),
  destroyTabCoordination: vi.fn(),
  isTabLeader: () => true,
  requestRefresh: () => Promise.resolve(null),
  onLeaderChange: () => () => {},
  onCrossTabLogout: () => () => {},
  broadcastLogout: vi.fn(),
}));

function makeAxiosError(status: number) {
  const err = new Error(`HTTP ${status}`) as Error & {
    isAxiosError: boolean;
    response: { status: number };
  };
  err.isAxiosError = true;
  err.response = { status };
  return err;
}

function makePasswordGateError(status: number, code?: string) {
  const err = new Error(`HTTP ${status}`) as Error & {
    isAxiosError: boolean;
    response: { status: number; data?: unknown };
  };
  err.isAxiosError = true;
  err.response = { status, data: code ? { error: { code } } : undefined };
  return err;
}

describe("auth-service", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  describe("login", () => {
    it("stores access token on success", async () => {
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: {
          access_token: "new-token",
          user: { id: 1, username: "admin", email: "admin@test.com" },
        },
      });

      const { login } = await import("../auth-service");
      const result = await login("admin", "password");

      expect(result).not.toBe(false);
      if (result) {
        expect(result.access_token).toBe("new-token");
        expect(result.user.username).toBe("admin");
      }
      expect(sessionStorage.getItem("access_token")).toBe("new-token");
    });

    it("merges permission claims from token when present", async () => {
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: {
          access_token:
            "eyJhbGciOiJIUzI1NiJ9.eyJwZXJtaXNzaW9ucyI6WyJhZG1pbiIsInVzZXIiXX0.abc",
          user: { id: 1, username: "admin" },
        },
      });

      const { login } = await import("../auth-service");
      const result = await login("admin", "password");

      expect(result).not.toBe(false);
      if (result) {
        expect(result.user.permissions).toEqual(["admin", "user"]);
      }
    });

    it("returns false on API failure", async () => {
      mockPost.mockRejectedValueOnce(new Error("Network error"));

      const { login } = await import("../auth-service");
      const result = await login("wrong", "wrong");

      expect(result).toBe(false);
    });
  });

  describe("logout", () => {
    it("clears session and redirects", async () => {
      mockPost.mockResolvedValueOnce({ status: 200 });
      sessionStorage.setItem("access_token", "will-be-cleared");

      const { logout } = await import("../auth-service");
      await logout();

      expect(sessionStorage.getItem("access_token")).toBeNull();
    });
  });

  describe("refreshAccessToken", () => {
    it("calls doRefresh and returns new token", async () => {
      mockPost.mockResolvedValueOnce({
        data: { access_token: "refreshed-token" },
      });

      const { refreshAccessToken } = await import("../auth-service");
      const result = await refreshAccessToken();

      expect(result).toBe("refreshed-token");
      expect(sessionStorage.getItem("access_token")).toBe("refreshed-token");
    });

    it("returns null on refresh failure", async () => {
      mockPost.mockRejectedValueOnce(new Error("Refresh failed"));

      const { refreshAccessToken } = await import("../auth-service");
      const result = await refreshAccessToken();

      expect(result).toBeNull();
    });
  });

  describe("proactive refresh", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it("startProactiveRefresh calls doRefresh on interval", async () => {
      sessionStorage.setItem("access_token", "some-token");
      mockPost.mockResolvedValue({ data: { access_token: "refreshed" } });

      const mod = await import("../auth-service");
      mod.startProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).toHaveBeenCalledWith("/refresh");
      mod.stopProactiveRefresh();
    });

    it("stops refresh timer when no token present", async () => {
      const mod = await import("../auth-service");
      mod.startProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).not.toHaveBeenCalled();
    });

    it("stopProactiveRefresh clears the timer", async () => {
      sessionStorage.setItem("access_token", "some-token");
      mockPost.mockResolvedValue({ data: { access_token: "refreshed" } });

      const mod = await import("../auth-service");
      mod.startProactiveRefresh();
      mod.stopProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).not.toHaveBeenCalled();
    });
  });

  describe("setupAxiosInterceptors", () => {
    it("registers response interceptor", async () => {
      const { setupAxiosInterceptors } = await import("../auth-service");
      const client = {
        interceptors: { response: { use: vi.fn() } },
      } as unknown as AxiosInstance;

      setupAxiosInterceptors(client);

      expect(client.interceptors.response.use).toHaveBeenCalled();
    });

    async function capturedErrorInterceptor() {
      const { setupAxiosInterceptors } = await import("../auth-service");
      const handlers: Array<(error: unknown) => Promise<never>> = [];
      const use = vi.fn(
        (
          _onFulfilled: unknown,
          onRejected: (error: unknown) => Promise<never>,
        ) => {
          handlers.push(onRejected);
        },
      );
      const client = {
        interceptors: { response: { use } },
      } as unknown as AxiosInstance;

      setupAxiosInterceptors(client);
      expect(handlers).toHaveLength(1);
      return handlers[0];
    }

    it("raises the blocking password gate on 428 and re-rejects", async () => {
      const interceptor = await capturedErrorInterceptor();
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      const store = useAuthStore();
      store.mustChangePassword = false;

      const gateError = makePasswordGateError(428, "PASSWORD_CHANGE_REQUIRED");

      await expect(interceptor(gateError)).rejects.toBe(gateError);
      expect(store.mustChangePassword).toBe(true);
    });

    it("does not raise the password gate for other failures", async () => {
      const interceptor = await capturedErrorInterceptor();
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      const store = useAuthStore();
      store.mustChangePassword = false;

      const serverError = makeAxiosError(500);

      await expect(interceptor(serverError)).rejects.toBe(serverError);
      expect(store.mustChangePassword).toBe(false);
    });
  });

  describe("checkAuth", () => {
    it("returns true when /validate succeeds", async () => {
      mockPost.mockResolvedValueOnce({ status: 200 });

      const { checkAuth } = await import("../auth-service");
      const result = await checkAuth();

      expect(result).toBe(true);
    });

    it("auto-refreshes on 401 and retries validation", async () => {
      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: "new-token" } })
        .mockResolvedValueOnce({ status: 200 });

      const { checkAuth } = await import("../auth-service");
      const result = await checkAuth();

      expect(result).toBe(true);
      expect(mockPost).toHaveBeenCalledWith("/refresh");
    });

    it("returns false when refresh fails after 401", async () => {
      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockRejectedValueOnce(new Error("Refresh failed"));

      const { checkAuth } = await import("../auth-service");
      const result = await checkAuth();

      expect(result).toBe(false);
    });

    it("returns false on non-401 axios error", async () => {
      mockPost.mockRejectedValueOnce(makeAxiosError(500));

      const { checkAuth } = await import("../auth-service");
      const result = await checkAuth();

      expect(result).toBe(false);
    });

    it("returns false on non-axios error", async () => {
      mockPost.mockRejectedValueOnce(new Error("Generic error"));

      const { checkAuth } = await import("../auth-service");
      const result = await checkAuth();

      expect(result).toBe(false);
    });
  });

  describe("restoreSession", () => {
    it("returns user when validate succeeds", async () => {
      sessionStorage.setItem("access_token", "valid-token");
      mockPost.mockResolvedValueOnce({
        data: { user: { id: 1, username: "admin" }, permissions: ["read"] },
      });

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.user?.username).toBe("admin");
        expect(result.user?.permissions).toEqual(["read"]);
      }
    });

    it("returns success false when no token in sessionStorage", async () => {
      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it("returns success false when validate returns no user", async () => {
      sessionStorage.setItem("access_token", "valid-token");
      mockPost.mockResolvedValueOnce({ data: {} });

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it("auto-refreshes on 401 and retries", async () => {
      sessionStorage.setItem("access_token", "stale-token");

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: "new-token" } })
        .mockResolvedValueOnce({
          data: {
            user: { id: 1, username: "refreshed" },
            permissions: ["write"],
          },
        });

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.user?.username).toBe("refreshed");
        expect(result.user?.permissions).toEqual(["write"]);
      }
    });

    it("returns success false when refresh fails after 401 on validate", async () => {
      sessionStorage.setItem("access_token", "stale-token");

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockRejectedValueOnce(new Error("fail"));

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it("returns success false when retry validate still fails after refresh", async () => {
      sessionStorage.setItem("access_token", "stale-token");

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: "new-token" } })
        .mockRejectedValueOnce(makeAxiosError(401));

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it("returns success false on non-401 error", async () => {
      sessionStorage.setItem("access_token", "some-token");
      mockPost.mockRejectedValueOnce(makeAxiosError(500));

      const { restoreSession } = await import("../auth-service");
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });
  });

  describe("password change required detection", () => {
    it("detects 428 with PASSWORD_CHANGE_REQUIRED code", async () => {
      const { isPasswordChangeRequired } = await import("../auth-service");
      expect(
        isPasswordChangeRequired(
          makePasswordGateError(428, "PASSWORD_CHANGE_REQUIRED"),
        ),
      ).toBe(true);
    });

    it("ignores 428 without the expected code", async () => {
      const { isPasswordChangeRequired } = await import("../auth-service");
      expect(isPasswordChangeRequired(makePasswordGateError(428))).toBe(false);
      expect(
        isPasswordChangeRequired(makePasswordGateError(428, "OTHER_CODE")),
      ).toBe(false);
    });

    it("ignores other statuses and non-axios errors", async () => {
      const { isPasswordChangeRequired } = await import("../auth-service");
      expect(isPasswordChangeRequired(makePasswordGateError(500))).toBe(false);
      expect(isPasswordChangeRequired(new Error("boom"))).toBe(false);
      expect(isPasswordChangeRequired(undefined)).toBe(false);
    });

    it("markPasswordChangeRequired sets the store flag", async () => {
      const { markPasswordChangeRequired } = await import("../auth-service");
      const { useAuthStore } = await import("../../stores/auth-store.svelte");

      useAuthStore().mustChangePassword = false;
      markPasswordChangeRequired();
      expect(useAuthStore().mustChangePassword).toBe(true);
    });
  });

  describe("changePassword", () => {
    it("posts credentials, stores the rotated token and clears the flag", async () => {
      const { changePassword } = await import("../auth-service");
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      const store = useAuthStore();
      store.mustChangePassword = true;
      sessionStorage.setItem("access_token", "old-token");

      mockPost.mockResolvedValueOnce({
        status: 200,
        data: { access_token: "rotated-token" },
      });

      const result = await changePassword("old-pass-1", "new-pass-123");

      expect(result.ok).toBe(true);
      expect(mockPost).toHaveBeenCalledWith(
        "/change-password",
        { current_password: "old-pass-1", new_password: "new-pass-123" },
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: "Bearer old-token",
          }),
        }),
      );
      expect(sessionStorage.getItem("access_token")).toBe("rotated-token");
      expect(store.mustChangePassword).toBe(false);
    });

    it("refreshes once and retries when the access token expired", async () => {
      const { changePassword } = await import("../auth-service");
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      const store = useAuthStore();
      store.mustChangePassword = true;
      sessionStorage.setItem("access_token", "stale-token");

      // 1) rotation attempt with the expired access token
      mockPost.mockRejectedValueOnce(makeAxiosError(401));
      // 2) refresh exchange
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: { access_token: "refreshed-token" },
      });
      // 3) retried rotation
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: { access_token: "rotated-token" },
      });

      const result = await changePassword("old-pass-1", "new-pass-123");

      expect(result.ok).toBe(true);
      expect(sessionStorage.getItem("access_token")).toBe("rotated-token");
      expect(store.mustChangePassword).toBe(false);
      expect(mockPost).toHaveBeenNthCalledWith(
        3,
        "/change-password",
        { current_password: "old-pass-1", new_password: "new-pass-123" },
        {
          headers: { Authorization: "Bearer refreshed-token" },
        },
      );
    });

    it("reports an expired session when the refresh token is dead", async () => {
      const { changePassword } = await import("../auth-service");
      const { labels } = await import("$shared/i18n");
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      useAuthStore().mustChangePassword = true;
      sessionStorage.setItem("access_token", "stale-token");

      mockPost.mockRejectedValueOnce(makeAxiosError(401));
      mockPost.mockRejectedValueOnce(makeAxiosError(401));

      const result = await changePassword("old-pass-1", "new-pass-123");

      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.message).toBe(labels.toastSessionExpired);
      // No third attempt: a dead refresh token is not recoverable here, the
      // modal stays open and the user can sign out.
      expect(mockPost).toHaveBeenCalledTimes(2);
      expect(useAuthStore().mustChangePassword).toBe(true);
    });

    it("returns the backend message on rejection", async () => {
      const { changePassword } = await import("../auth-service");
      const { useAuthStore } = await import("../../stores/auth-store.svelte");
      useAuthStore().mustChangePassword = true;

      const err = new Error("HTTP 400") as Error & {
        isAxiosError: boolean;
        response: { status: number; data?: unknown };
      };
      err.isAxiosError = true;
      err.response = {
        status: 400,
        data: { error: { message: "password too weak" } },
      };
      mockPost.mockRejectedValueOnce(err);

      const result = await changePassword("old-pass-1", "short");

      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.message).toBe("password too weak");
      expect(useAuthStore().mustChangePassword).toBe(true);
    });
  });
});
