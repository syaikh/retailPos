/**
 * Behavioral tests for cross-tab refresh coordination.
 *
 * Leader election and waiter settlement are module-level state, so each test
 * re-imports a fresh module (vi.resetModules) and drives the message handler
 * through a fake BroadcastChannel — there is no second real tab to answer.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { mockPost } = vi.hoisted(() => ({
  mockPost: vi.fn(),
}));

vi.mock("axios", () => ({
  default: { create: vi.fn(() => ({ post: mockPost })) },
}));

type CoordinationModule = typeof import("../tab-coordination");

/**
 * Minimal BroadcastChannel: posts reach every *other* live instance on the
 * same channel name, which is the guarantee the real API gives. Tests hand
 * the module its inbound messages directly via receive().
 */
class FakeBroadcastChannel {
  static instances: FakeBroadcastChannel[] = [];
  readonly posted: unknown[] = [];
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  private closed = false;

  constructor(readonly name: string) {
    FakeBroadcastChannel.instances.push(this);
  }

  postMessage(data: unknown): void {
    if (this.closed) return;
    this.posted.push(data);
    for (const peer of FakeBroadcastChannel.instances) {
      if (peer !== this && !peer.closed) peer.onmessage?.({ data });
    }
  }

  close(): void {
    this.closed = true;
    this.onmessage = null;
  }

  /** Deliver a message as if another tab had broadcast it. */
  receive(data: unknown): void {
    this.onmessage?.({ data });
  }
}

let mod: CoordinationModule;

/** The channel the module most recently opened. */
function channel(): FakeBroadcastChannel {
  const inst = FakeBroadcastChannel.instances.at(-1);
  if (!inst) throw new Error("module never opened a channel");
  return inst;
}

/** Run the election to completion with no competing tab. */
async function becomeLeader(): Promise<void> {
  mod.initTabCoordination();
  // ELECTION_TIMEOUT is 500ms; nothing answers the initial PING.
  await vi.advanceTimersByTimeAsync(600);
}

beforeEach(async () => {
  vi.useFakeTimers();
  FakeBroadcastChannel.instances = [];
  vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  sessionStorage.clear();
  mockPost.mockReset();
  mockPost.mockResolvedValue({ data: { access_token: "leader-refreshed" } });
  vi.resetModules();
  mod = await import("../tab-coordination");
});

afterEach(() => {
  mod.destroyTabCoordination();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("tab-coordination", () => {
  it("assumes leadership before an election has run", () => {
    expect(mod.isTabLeader()).toBe(true);
  });

  it("yields leadership when another tab answers the election", async () => {
    mod.initTabCoordination();
    channel().receive({ type: "PONG", tabId: "leader-1" });

    expect(mod.isTabLeader()).toBe(false);

    // The election timeout that follows must not reclaim leadership.
    await vi.advanceTimersByTimeAsync(600);
    expect(mod.isTabLeader()).toBe(false);
  });

  it("settles concurrent follower refreshes from a single result", async () => {
    mod.initTabCoordination();
    channel().receive({ type: "PONG", tabId: "leader-1" });

    const first = mod.requestRefresh();
    const second = mod.requestRefresh();
    channel().receive({ type: "REFRESH_RESULT", token: "fresh-token" });

    await expect(first).resolves.toBe("fresh-token");
    await expect(second).resolves.toBe("fresh-token");
    // The follower persists the leader's token: sessionStorage is per-tab.
    expect(sessionStorage.getItem("access_token")).toBe("fresh-token");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("rejects follower waiters the leader never answers", async () => {
    mod.initTabCoordination();
    channel().receive({ type: "PONG", tabId: "leader-1" });

    const pending = mod.requestRefresh();
    await vi.advanceTimersByTimeAsync(10_000);

    await expect(pending).rejects.toThrow(
      "Timed out waiting for the leader tab to refresh",
    );
    expect(sessionStorage.getItem("access_token")).toBeNull();
  });

  it("rejects follower waiters when the leader refresh fails", async () => {
    mod.initTabCoordination();
    channel().receive({ type: "PONG", tabId: "leader-1" });

    const pending = mod.requestRefresh();
    channel().receive({ type: "REFRESH_FAILED" });

    await expect(pending).rejects.toThrow("Refresh failed on leader");
    expect(sessionStorage.getItem("access_token")).toBeNull();
  });

  it("rejects pending waiters when coordination is torn down", async () => {
    mod.initTabCoordination();
    channel().receive({ type: "PONG", tabId: "leader-1" });

    const pending = mod.requestRefresh();
    mod.destroyTabCoordination();

    await expect(pending).rejects.toThrow("Tab coordination was torn down");
  });

  it("refreshes on behalf of followers once it is leader", async () => {
    await becomeLeader();
    expect(mod.isTabLeader()).toBe(true);
    // The leader refreshes itself: null means "do it locally".
    await expect(mod.requestRefresh()).resolves.toBeNull();

    mockPost.mockResolvedValueOnce({ data: { access_token: "leader-token" } });
    channel().receive({ type: "REFRESH_REQUEST", tabId: "follower-1" });
    await Promise.resolve();

    expect(mockPost).toHaveBeenCalledWith("/refresh");
    // Persisted here too: broadcasting alone would leave this tab stale.
    expect(sessionStorage.getItem("access_token")).toBe("leader-token");
    expect(channel().posted).toContainEqual({
      type: "REFRESH_RESULT",
      token: "leader-token",
    });
  });

  it("broadcasts REFRESH_FAILED when its own refresh fails", async () => {
    await becomeLeader();
    mockPost.mockRejectedValueOnce(new Error("refresh rejected"));

    channel().receive({ type: "REFRESH_REQUEST", tabId: "follower-1" });
    await Promise.resolve();

    expect(mockPost).toHaveBeenCalledTimes(1);
    expect(channel().posted).toContainEqual({ type: "REFRESH_FAILED" });
  });
});
