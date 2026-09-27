import { describe, it, expect, beforeEach, vi } from "vitest";

function freshImport() {
  vi.resetModules();
  return import("../printConfig.svelte");
}

describe("PrintConfigStore", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it("defaults to preview mode and the dev agent URL", async () => {
    const { printConfig } = await import("../printConfig.svelte");
    expect(printConfig.mode).toBe("preview");
    expect(printConfig.agentUrl).toBe("http://localhost:9123");
  });

  it("setMode updates and persists to localStorage", async () => {
    const { printConfig } = await freshImport();
    printConfig.setMode("silent");
    expect(printConfig.mode).toBe("silent");
    const raw = localStorage.getItem("pos.printConfig");
    expect(raw).toBeTruthy();
    expect(JSON.parse(raw as string).mode).toBe("silent");
  });

  it("toggleMode flips between preview and silent", async () => {
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe("preview");
    printConfig.toggleMode();
    expect(printConfig.mode).toBe("silent");
    printConfig.toggleMode();
    expect(printConfig.mode).toBe("preview");
  });

  it("setAgentUrl trims surrounding whitespace and persists; empty falls back to default", async () => {
    const { printConfig } = await freshImport();
    printConfig.setAgentUrl("  http://example:9000/  ");
    expect(printConfig.agentUrl).toBe("http://example:9000/");
    printConfig.setAgentUrl("   ");
    expect(printConfig.agentUrl).toBe("http://localhost:9123");
  });

  it("hydrates mode and agentUrl from localStorage", async () => {
    localStorage.setItem(
      "pos.printConfig",
      JSON.stringify({ mode: "silent", agentUrl: "http://shop:1234" }),
    );
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe("silent");
    expect(printConfig.agentUrl).toBe("http://shop:1234");
  });

  it("ignores malformed localStorage JSON and keeps defaults", async () => {
    localStorage.setItem("pos.printConfig", "{not valid json");
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe("preview");
    expect(printConfig.agentUrl).toBe("http://localhost:9123");
  });

  it("seeds mode from VITE_PRINT_MODE env when storage is empty", async () => {
    vi.stubEnv("VITE_PRINT_MODE", "silent");
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe("silent");
  });

  it("locks mode to silent when VITE_PRINT_MODE=silent and ignores a stored preview preference", async () => {
    localStorage.setItem(
      "pos.printConfig",
      JSON.stringify({ mode: "preview", agentUrl: "http://shop:1234" }),
    );
    vi.stubEnv("VITE_PRINT_MODE", "silent");
    const { printConfig } = await freshImport();
    expect(printConfig.locked).toBe(true);
    expect(printConfig.mode).toBe("silent");
  });

  it("does not lock when VITE_PRINT_MODE is not silent", async () => {
    vi.stubEnv("VITE_PRINT_MODE", "preview");
    const { printConfig } = await freshImport();
    expect(printConfig.locked).toBe(false);
    expect(printConfig.mode).toBe("preview");
  });

  it("setMode is a no-op while locked (cannot revert to preview)", async () => {
    vi.stubEnv("VITE_PRINT_MODE", "silent");
    const { printConfig } = await freshImport();
    printConfig.setMode("preview");
    expect(printConfig.mode).toBe("silent");
  });

  it("toggleMode is a no-op while locked", async () => {
    vi.stubEnv("VITE_PRINT_MODE", "silent");
    const { printConfig } = await freshImport();
    printConfig.toggleMode();
    expect(printConfig.mode).toBe("silent");
  });
});

describe("isLocalAgentUrl", () => {
  it("treats loopback addresses as local", async () => {
    const { isLocalAgentUrl } = await freshImport();
    for (const url of [
      "http://localhost:9123",
      "http://localhost:8080",
      "https://LocalHost:9123",
      "http://127.0.0.1:9123",
      "http://[::1]:9123",
      "http://localhost:9123/",
    ]) {
      expect(isLocalAgentUrl(url), url).toBe(true);
    }
  });

  it("treats a LAN or remote address as not local", async () => {
    const { isLocalAgentUrl } = await freshImport();
    for (const url of [
      "http://192.168.1.10:9123",
      "http://192.168.1.50:9123",
      "http://printer-server.local:9123",
      "https://pos.example.com",
    ]) {
      expect(isLocalAgentUrl(url), url).toBe(false);
    }
  });

  it("does not treat a lookalike host as local", async () => {
    // A substring check on "localhost" would wrongly accept all of these,
    // which would silence the warning exactly when it is needed.
    const { isLocalAgentUrl } = await freshImport();
    for (const url of [
      "http://localhost.evil.example:9123",
      "http://notlocalhost:9123",
      "http://127.0.0.1.evil.example:9123",
    ]) {
      expect(isLocalAgentUrl(url), url).toBe(false);
    }
  });

  it("treats an unparseable or empty URL as not local", async () => {
    const { isLocalAgentUrl } = await freshImport();
    for (const url of ["", "   ", "localhost:9123", "not a url"]) {
      expect(isLocalAgentUrl(url), url).toBe(false);
    }
  });
});
