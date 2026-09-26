import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const markPasswordChangeRequired = vi.fn();
const refreshAccessToken = vi.fn();
const logout = vi.fn();
const setupAxiosInterceptors = vi.fn();
const getAuthToken = vi.fn((): string | null => null);

vi.mock("$modules/auth", () => ({
  getAuthToken: () => getAuthToken(),
  refreshAccessToken: () => refreshAccessToken(),
  setupAxiosInterceptors: () => setupAxiosInterceptors(),
  logout: () => logout(),
  markPasswordChangeRequired: () => markPasswordChangeRequired(),
  PASSWORD_CHANGE_REQUIRED: "PASSWORD_CHANGE_REQUIRED",
}));

vi.mock("axios", () => ({
  default: {
    create: () => ({
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    }),
  },
}));

function makeResponse(status: number, body: unknown): Response {
  return {
    status,
    clone: () => ({ json: () => Promise.resolve(body) }),
    json: () => Promise.resolve(body),
  } as unknown as Response;
}

describe("apiFetch", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    getAuthToken.mockReturnValue(null);
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns the response untouched on success", async () => {
    const { apiFetch } = await import("../http-client");
    const ok = makeResponse(200, { data: [] });
    fetchMock.mockResolvedValueOnce(ok);

    const res = await apiFetch("/api/stores");

    expect(res).toBe(ok);
    expect(markPasswordChangeRequired).not.toHaveBeenCalled();
  });

  it("raises the password-change flag on a 428 PASSWORD_CHANGE_REQUIRED", async () => {
    const { apiFetch } = await import("../http-client");
    const gated = makeResponse(428, {
      error: { code: "PASSWORD_CHANGE_REQUIRED", message: "rotate" },
    });
    fetchMock.mockResolvedValueOnce(gated);

    const res = await apiFetch("/api/products");

    expect(res).toBe(gated);
    expect(markPasswordChangeRequired).toHaveBeenCalledTimes(1);
  });

  it("ignores a 428 with a different code", async () => {
    const { apiFetch } = await import("../http-client");
    fetchMock.mockResolvedValueOnce(
      makeResponse(428, { error: { code: "OTHER" } }),
    );

    await apiFetch("/api/products");

    expect(markPasswordChangeRequired).not.toHaveBeenCalled();
  });

  it("ignores an unparseable 428 body", async () => {
    const { apiFetch } = await import("../http-client");
    const broken = {
      status: 428,
      clone: () => ({ json: () => Promise.reject(new Error("bad json")) }),
    } as unknown as Response;
    fetchMock.mockResolvedValueOnce(broken);

    await apiFetch("/api/products");

    expect(markPasswordChangeRequired).not.toHaveBeenCalled();
  });

  it("refreshes and retries once on 401", async () => {
    getAuthToken.mockReturnValue("stale-token");
    refreshAccessToken.mockResolvedValueOnce("fresh-token");
    const { apiFetch } = await import("../http-client");
    fetchMock
      .mockResolvedValueOnce(makeResponse(401, {}))
      .mockResolvedValueOnce(makeResponse(200, { ok: true }));

    const res = await apiFetch("/api/pos/cart");

    expect(res.status).toBe(200);
    expect(refreshAccessToken).toHaveBeenCalledTimes(1);
    expect(logout).not.toHaveBeenCalled();
    const retryHeaders = fetchMock.mock.calls[1][1].headers;
    expect(retryHeaders.Authorization).toBe("Bearer fresh-token");
  });

  it("logs out and throws when the 401 refresh fails", async () => {
    getAuthToken.mockReturnValue("stale-token");
    refreshAccessToken.mockResolvedValueOnce(null);
    const { apiFetch } = await import("../http-client");
    fetchMock.mockResolvedValueOnce(makeResponse(401, {}));

    await expect(apiFetch("/api/pos/cart")).rejects.toThrow("Session expired");
    expect(logout).toHaveBeenCalledTimes(1);
  });
});
