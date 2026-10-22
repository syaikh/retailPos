# Codebase Explore Agent

This guide explains how to configure and use the explore subagent for codebase exploration with the `open-codebase-index` plugin.

## Overview

The explore agent gives a subagent access to the `open-codebase-index` tools for repository navigation beyond plain grep/glob. It replaces the built-in `explore` subagent.

**The index in this project runs in `structural` mode** (`.opencode/codebase-index.json` → `indexing.mode: "structural"`): BM25 keyword search, a symbol catalog, and a call graph — **no embeddings, no embedding provider, no Ollama**. Consequences:

- Queries are lexical. Describe behavior in the words used in the code, or cross-check with the Semble MCP server (`semble.search`) for paraphrase-style questions.
- Two tools are unavailable because they need vectors: **`find_similar`** (returns a non-retryable `UNSUPPORTED_OPERATION` error) and **`architecture_context`** (throws `vector store is unavailable in structural indexing mode`). Use `semble.find_related` / `semble.search` for similarity, and `codebase_context` or `code_communities` for repository orientation.

## Table of Contents

- [Available Tools](#available-tools)
- [Configuration](#configuration)
- [Usage](#usage)
- [Reverting to Built-in Explore](#reverting-to-built-in-explore)
- [Troubleshooting](#troubleshooting)
- [Verified Behavior](#verified-behavior)

---

## Available Tools

| Tool | Description |
|------|-------------|
| `codebase_context` | Route repository questions to bounded evidence packs (lexical/structural) |
| `codebase_search` | Search for matching code and return full source content |
| `codebase_peek` | Find likely locations with metadata only (file, line, name) |
| `implementation_lookup` | Find a symbol's authoritative definition (argument: `query`) |
| `call_graph` | Query direct callers or callees (exact symbol name + `filePath` on ambiguity) |
| `call_graph_path` | Find shortest connection path between two symbols |
| `codebase_edit_context` | Get bounded context before modifying code |
| `code_communities` | Discover module boundaries and hub symbols |
| `pr_impact` | Analyze blast radius of a branch (branch vs. base — reports zero for uncommitted work) |
| `index_codebase` | Incremental indexing; structural mode requests no embeddings |
| `index_status` | Check index readiness, mode, provider, and pending migrations |
| `index_health_check` | Check index health and drop stale entries after deletions |
| `add_knowledge_base` / `list_knowledge_bases` / `remove_knowledge_base` | Manage documentation folders added to the index |
| `read`, `glob`, `grep` | Base file tools |

**Unavailable in this project:** `find_similar`, `architecture_context` (both require vectors).

**Second index (Semble MCP):** `semble.search` and `semble.find_related`, reached inside `execute` as `tools.semble["search"]` and `tools.semble["find_related"]`. Semble indexes `docs/` too, so ignore hits under `docs/`, `*.md`, and `database/migrations/` when judging whether something is real code. Semble has no call graph — keep using `call_graph` / `call_graph_path`.

---

## Configuration

The explore agent is configured in `.opencode/opencode.json` with a custom prompt:

```json
{
  "agent": {
    "explore": {
      "description": "Semantic codebase exploration using codebase index tools",
      "mode": "subagent",
      "prompt": "{file:./prompts/explore.txt}",
      "permission": { "codebase_context": "allow", "...": "..." }
    }
  }
}
```

The prompt file (`.opencode/prompts/explore.txt`) holds the tool-selection guide, the `call_graph` exact-name/ambiguity workflow, and the fallback rules.

### How It Works

- The `agent.explore` entry overrides the built-in explore subagent.
- The `permission` block grants access to the plugin's index tools.
- The agent runs in a child session with its own context.
- **Agent definitions are cached for the life of an opencode session.** After editing this config, restart opencode. If an already-running `explore` session shows only `glob`, `grep`, `read`, `shell`, `webfetch`, `websearch`, it is stale — restart, do not edit the config further.

---

## Usage

### Via @mention

```
@explore use codebase_context to find how authentication is structured
```

### Via the `subagent` tool

```javascript
subagent({
  agent: "explore",
  description: "Map authentication flow",
  prompt: "Use codebase_context to find how authentication is structured"
})
```

(Older examples showing `task({ subagent_type: "explore" })` refer to a removed API.)

For heavy research fan-out that also needs the index, the `general` agent carries the same toolset.

### Example Queries

- `@explore use codebase_context to find how authentication is structured`
- `@explore use implementation_lookup to find the AuthService definition`
- `@explore use call_graph with filePath to find all callers of CreateSale`
- `@explore use codebase_search for error handling patterns, then grep for specific error codes`

---

## Reverting to Built-in Explore

Remove the `agent.explore` block from `.opencode/opencode.json`:

```json
{
  "agent": {}
}
```

Restart opencode for changes to take effect.

---

## Troubleshooting

### "Unknown agent type" error

Agent configuration is loaded at session start. Restart opencode after config changes:

```
# Exit opencode, then:
opencode
```

### Explore agent has no index tools

Run the probe again after a restart. If it still lists only base tools, verify the plugin is wired up:

```bash
# Plugin lives inside the project config directory
ls .opencode/node_modules/open-codebase-index/

# Index config
cat .opencode/codebase-index.json
```

### `find_similar` / `architecture_context` errors

Expected in structural mode — those tools need vectors. Use `semble.find_related` / `semble.search` instead.

### Index not ready or stale

Use the `/status` slash command (or the `index_status` / `index_health_check` tools).

To refresh: `/index` (incremental) — and only when files actually changed.

**Never run `/reindex` or a forced `index_codebase` on your own initiative**: it is expensive and the user runs it. Report staleness instead, and fall back to `grep`/`read` meanwhile.

### Diagnostics are disabled by default

- `index_logs` replies "Debug mode is disabled" until `"debug": { "enabled": true }` is set.
- `index_metrics` replies "Metrics collection is disabled" until `"effectivenessMetrics": { "enabled": true }` is set.

---

## Verified Behavior

Last verified **2026-10-21** against the structural index (`open-codebase-index` 0.37.1):

| Tool | Status | Notes |
|------|--------|-------|
| `codebase_context` | ✅ | Bounded evidence packs with relevance scores |
| `implementation_lookup` | ✅ | Takes `query`, not `symbol` |
| `codebase_search` | ✅ | `directory` filter + `minScore` (0.4) can drop low-scoring matches — a "No matching code found" reply is not necessarily a missing file |
| `codebase_peek` | ✅ | Metadata-only navigation; deep files are indexed (`maxDepth: 5` does not exclude them) |
| `call_graph` / `call_graph_path` | ⚠️ | Exact symbol names, and duplicate names (test mocks, overloads) require `filePath` / `fromFilePath` / `toFilePath` |
| `pr_impact` | ⚠️ | Diffs the branch against its base — reports `Files changed: 0` for uncommitted work |
| `find_similar` | ❌ | Unavailable in structural mode |
| `architecture_context` | ❌ | Throws `vector store is unavailable in structural indexing mode` |

### Usage Notes

- Check the working tree (`git status`, `git diff --cached`) before trusting `pr_impact`.
- Use `grep` for exhaustive literal matches — semantic search returns top N only.
- Report index results as `file_path:line_number`.

---

## Related Documentation

- Tool-selection table in `AGENTS.md` ("Codebase Investigation & Semantic Index")
- [open-codebase-index documentation](https://github.com/Helweg/open-codebase-index)
- [OpenCode Agents Documentation](https://opencode.ai/docs/agents/)
