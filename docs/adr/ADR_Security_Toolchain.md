# ADR: Go Security Toolchain (gosec, gitleaks, fuzzing)

| Field | Value |
|-------|-------|
| Status | **Accepted** |
| Date | 2026-10-18 |
| Deciders | Tim Engineering |
| Scope | Go security scanning beyond govulncheck: first-party SAST, secret detection, fuzz testing |
| Out of scope | Frontend SAST, runtime protections (WAF, RASP), dependency update policy |
| Related | `.golangci.yaml`, `.pre-commit-config.yaml`, `.github/workflows/ci.yml`, `.github/workflows/fuzz.yml` |

---

## 1. Context

CI already ran `govulncheck` (dependency CVEs), `golangci-lint` (correctness linters: govet,
staticcheck, errcheck, ineffassign, unused, revive, depguard), and CodeQL. That stack had three
gaps:

1. **First-party SAST** — no tool scanned our own source for structural flaws govulncheck
   cannot see: hardcoded credentials, SQL built by string concatenation, unsafe file access,
   weak randomness, permissive file modes.
2. **Secret detection** — nothing stopped API keys, tokens, or credentials from being committed.
3. **Fuzz testing** — no coverage of input-handling edge cases (pricing, parsing, shared
   validators) beyond hand-written unit tests.

## 2. Decision

Adopt four complementary tools, each with a distinct focus:

| Tool | Focus | Where it runs |
|------|-------|---------------|
| `govulncheck` (existing) | Exploitable CVEs in dependencies | CI: `Go: Vulnerability Check` |
| `gosec` | First-party source: hardcoded secrets, SQL injection (G202/G304), weak crypto/randomness (G404), file permissions (G301/G306), int conversions (G115) | Enabled inside `golangci-lint` via `.golangci.yaml` — reuses the existing `Go: Lint` job, no parallel lint run |
| `gitleaks` | Committed secrets (keys, tokens, passwords) | Pre-commit hook (`.pre-commit-config.yaml`, optional locally) + CI `Secrets: Gitleaks` job with `fetch-depth: 0` (full history scan) |
| `go test -fuzz` | Input-handling edge cases static analysis cannot foresee | `.github/workflows/fuzz.yml` — weekly schedule + `workflow_dispatch`, `-fuzztime 5m` per package, **non-blocking** |

`staticcheck` was already enabled in golangci-lint; no change needed there.

### 2.1 Rationale for gosec inside golangci-lint

Running gosec as a standalone CI job would duplicate checkout/setup and create a second config
surface. golangci-lint natively hosts gosec, so enabling it in `.golangci.yaml` adds SAST to the
job that already runs on every push/PR.

### 2.2 Rationale for non-blocking fuzzing

Fuzzing is non-deterministic and time-boxed. Blocking PRs on it would slow the pipeline for
marginal per-PR signal. A scheduled run finds the same crashes overnight; `workflow_dispatch`
allows on-demand runs. Targeted packages today: `./internal/pricing/...`, `./internal/shared/...`
— expand as `Fuzz*` tests are added.

## 3. Consequences

### 3.1 Baseline remediation (commits `d46883d4`, `0ef77f8b`)

Enabling gosec surfaced **230 findings** (227 server + 3 print-agent), all cleared in
`0ef77f8b`. Notable root-cause fixes rather than suppressions:

- `cmd/dummy` seeder (197 findings):
  - `generateAuditLogs` built SQL by string concatenation (G202) — rewritten to bind all 11
    columns as query parameters.
  - Hardcoded DSN credentials (G101) — now composed from `DATABASE_URL` or the `DB_*` env vars
    the backend already reads.
  - `math/rand` (G404) — replaced with crypto/rand-backed helpers in `cmd/dummy/rand.go`.
- `internal/platform/importexport`: `context.Background()` → `context.WithoutCancel(ctx)` (G115)
  so work that outlives the request keeps trace values.
- File access (G304): `os.ReadFile` on joined paths replaced with `fs.ReadFile(os.DirFS(...), name)`
  / `fs.Glob`; remaining direct writes use `filepath.Clean`.
- File modes: directories `0755 → 0750`, sensitive files (logos, receipts, fixtures) `0644 → 0600`
  (G301/G306), including `tools/print-agent`.
- Overflow guards before narrowing `int → int32` (G115); masked byte conversions in print-agent.
- Test fixtures: cookie literals gain `HttpOnly/Secure/SameSite` (G124); secret-looking
  const names renamed (G101).

**One documented suppression remains:** `internal/config/config_test.go` — `exec.Command(os.Args[0], ...)`
re-exec idiom (G204). No code shape satisfies the rule: the command path must be the test
binary's own path. Justification is stated inline at the `//nolint` site.

### 3.2 Behavior changes to be aware of

- `podman-deploy.sh` passes `DB_*` vars to the seeder; the seeder now honors its own
  `DB_PORT` instead of silently hardcoding 5433 (matches backend env names).
- Seeder randomness is crypto-backed — same distributions, different source.
- `gitleaks` scans full history on every push/PR; a historical secret will fail
  `Secrets: Gitleaks` until rotated and purged.

### 3.3 Ongoing policy

- New/edited Go code must be gosec-clean on the next CI run (same rule as the other linters
  in AGENTS.md).
- `//nolint:gosec` requires an inline justification; no rule exclusions or severity downgrades
  in `.golangci.yaml` as a "fix".
- Adding `Fuzz*` tests to parser/validator/calculation packages is encouraged; the scheduled
  workflow picks them up automatically (`-fuzz=Fuzz` matches any fuzz target in the package).
- Install the pre-commit hook with `pre-commit install` (optional, but recommended — catches
  secrets before they reach history).

## 4. References

- Commits: `d46883d4` (toolchain adoption), `0ef77f8b` (gosec remediation)
- Config: `.golangci.yaml` (`linters.enable`), `.pre-commit-config.yaml`,
  `.github/workflows/ci.yml` (`secrets-check` job), `.github/workflows/fuzz.yml`
- Tool docs: [gosec](https://github.com/securego/gosec),
  [gitleaks](https://github.com/gitleaks/gitleaks),
  [Go fuzzing](https://go.dev/doc/testing/fuzzing)
