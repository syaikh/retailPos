# Semantic Explore Agent

This guide explains how to configure and use the semantic explore subagent for codebase exploration using the `open-codebase-index` plugin.

## Overview

The semantic explore agent provides access to the `open-codebase-index` tools, enabling semantic search and codebase analysis beyond basic grep/glob operations. It replaces the built-in `explore` subagent with enhanced capabilities.

## Table of Contents

- [Available Tools](#available-tools)
- [Configuration](#configuration)
- [Usage](#usage)
- [Reverting to Built-in Explore](#reverting-to-built-in-explore)
- [Troubleshooting](#troubleshooting)

---

## Available Tools

The semantic explore agent has access to the following tools:

| Tool | Description |
|------|-------------|
| `codebase_context` | Route repository questions to bounded evidence packs |
| `codebase_search` | Search by meaning and return full source results |
| `codebase_peek` | Find likely locations with minimal content |
| `implementation_lookup` | Find a symbol's authoritative definition |
| `call_graph` | Query direct callers or callees |
| `call_graph_path` | Find shortest connection path between symbols |
| `find_similar` | Find code similar to a given snippet |
| `codebase_edit_context` | Get bounded context before modifying code |
| `pr_impact` | Analyze blast radius of a branch or change |
| `code_communities` | Discover module boundaries and hub symbols |
| `index_codebase` | Index the codebase for semantic search |
| `index_status` | Check index readiness |
| `add_knowledge_base` | Add folders to the index |
| `list_knowledge_bases` | List configured knowledge bases |
| `remove_knowledge_base` | Remove a knowledge base |
| `read` | Read file contents |
| `glob` | Find files by pattern |
| `grep` | Search file contents |

---

## Configuration

The semantic explore agent is configured in `.opencode/opencode.json` with a custom prompt:

```json
{
  "agent": {
    "explore": {
      "description": "Semantic codebase exploration using codebase index tools",
      "mode": "subagent",
      "prompt": "{file:./prompts/explore.txt}",
      "permission": { ... }
    }
  }
}
```

The prompt file (`.opencode/prompts/explore.txt`) contains tool selection guidance and the `call_graph` usage workflow.
```

### How It Works

- The `agent.explore` entry overrides the built-in explore subagent
- The `permission` block grants access to semantic tools from the `open-codebase-index` plugin
- The agent runs in a child session with its own context

---

## Usage

### Via @mention

Type `@explore` followed by your query:

```
@explore use codebase_context to find how authentication is structured
```

### Via Task Tool (Programmatic)

```javascript
task({
  subagent_type: "explore",
  prompt: "Use codebase_context to find how authentication is structured"
})
```

### Example Queries

**Semantic queries (preferred):**
- `@explore use codebase_context to find how authentication is structured`
- `@explore use implementation_lookup to find the AuthService definition`
- `@explore use call_graph to find all callers of the Login method`

**Hybrid queries:**
- `@explore use codebase_search to find all error handling patterns, then grep for specific error codes`

---

## Reverting to Built-in Explore

To restore the original built-in explore agent, remove the `agent.explore` block from `.opencode/opencode.json`:

```json
{
  "agent": {
    "explore": <--- delete this entire block
  }
}
```

Restart opencode for changes to take effect.

---

## Troubleshooting

### "Unknown agent type" error

The agent configuration is loaded at session start. Restart opencode after making config changes:

```
# Exit opencode, then:
opencode
```

### Semantic tools not available

Verify the `open-codebase-index` plugin is installed and configured:

```bash
# Check plugin exists
ls ~/.config/opencode/node_modules/open-codebase-index/

# Check project config
cat .opencode/codebase-index.json
```

### Index not ready

Check index status:

```
@index_status
```

If the index is not ready, run:

```
@index_codebase
```

---

## Test Results

Verified on 2026-09-04 with `open-codebase-index` plugin:

| Tool | Status | Notes |
|------|--------|-------|
| `codebase_context` | ✅ PASS | Returns deduplicated evidence packs with relevance scores |
| `implementation_lookup` | ✅ PASS | Precise definition lookup (score 0.99) |
| `codebase_search` | ✅ PASS | Returns full source content |
| `call_graph` | ⚠️ FAIL | Requires exact symbol names (e.g., `AuthService.Login`, not just `Login`) |
| `codebase_peek` | ✅ PASS | Returns lightweight metadata for navigation |

### Usage Notes

- `call_graph` requires **exact symbol names**. Use `implementation_lookup` first to find the correct symbol name, then pass it to `call_graph`.
- `codebase_context` is the best starting point for exploratory queries — it routes questions to bounded evidence packs.
- `implementation_lookup` is best for finding a specific symbol's definition.

---

## Related Documentation

- [open-codebase-index documentation](https://github.com/Helweg/open-codebase-index)
- [OpenCode Agents Documentation](https://opencode.ai/docs/agents/)
