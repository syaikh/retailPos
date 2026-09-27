import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "PrintModeToggle.svelte"),
    "utf-8",
  );
}

describe("PrintModeToggle.svelte source-structure guards", () => {
  const src = getSource();

  it("imports printConfig store", () => {
    // Matched with a regex rather than an exact string: prettier decides where
    // to wrap the import, and this guard should not fail when it reflows.
    expect(src).toMatch(
      /import\s*\{\s*printConfig\s*,\s*isLocalAgentUrl\s*,?\s*\}\s*from\s*"\$shared\/stores\/printConfig\.svelte"/,
    );
  });

  it("imports i18n labels", () => {
    expect(src).toContain('import { labels } from "$shared/i18n"');
  });

  it("renders preview and silent segmented controls", () => {
    expect(src).toContain("{labels.preview}");
    expect(src).toContain("{labels.silent}");
  });

  it("hides the segmented controls when printConfig.locked and shows a silent badge", () => {
    expect(src).toContain("{#if !printConfig.locked}");
    expect(src).toContain("{:else}");
  });

  it('binds preview mode to setMode("preview")', () => {
    expect(src).toContain('printConfig.setMode("preview")');
  });

  it('binds silent mode to setMode("silent")', () => {
    expect(src).toContain('printConfig.setMode("silent")');
  });

  it("has an editor toggle (gear) for agent settings", () => {
    expect(src).toContain("function openEditor");
  });

  it("saves the agent URL via setAgentUrl", () => {
    expect(src).toContain("printConfig.setAgentUrl(urlDraft)");
  });

  it("tests the agent connection via /health", () => {
    expect(src).toContain("function testConnection");
    expect(src).toContain("/health");
  });

  it("shows agent connection status labels", () => {
    expect(src).toContain("{labels.agentConnected}");
    expect(src).toContain("{labels.agentUnreachable}");
  });

  it("warns when silent printing is pointed at an agent on another machine", () => {
    expect(src).toContain("agentIsRemote");
    expect(src).toContain("isLocalAgentUrl(printConfig.agentUrl)");
    expect(src).toContain("{labels.agentNotLocalhost}");
  });

  it("only warns in silent mode, where the agent is actually used", () => {
    // In preview mode the browser print dialog is used and the agent is never
    // contacted, so a remote address is not a misconfiguration worth flagging.
    expect(src).toMatch(/printConfig\.mode === "silent" &&\s*!isLocalAgentUrl/);
  });
});
