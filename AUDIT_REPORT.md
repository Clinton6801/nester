# Nester — Post-Campaign Repository Audit

**Date:** 2026-07-04 · **Branch:** `fix/661-txhistory` · **Scope:** whole monorepo, breadth-first (980 tracked files)
**Mode:** report only — no code changes applied. Fixes await approval.

> Companion to the existing maintainer queue in [`OSS_CLEANUP.md`](OSS_CLEANUP.md) and the
> [`CI_CD_VALIDATION_REPORT.md`](CI_CD_VALIDATION_REPORT.md). This report is the categorized,
> severity-rated superset. Where an item already lives in `OSS_CLEANUP.md` it is cross-referenced,
> not duplicated.

## Repository shape

| Stack | Location | Files | Notes |
|---|---|---|---|
| Go API (primary) | `apps/api` | 378 | Full domain set, `cmd/api/main.go` (~665 LoC), 109 `_test.go` |
| Frontend dApp | `apps/dapp` | 221 | Next.js + Soroban contract bindings; all 17 TS tests here |
| Marketing site | `apps/website` | 71 | Next.js, 0 tests |
| Contracts | `packages/contracts` | 58 | Rust/Soroban, has `test.rs` |
| Go API (duplicate) | `services/api` | 18 | **Competing/abandoned** skeleton |
| Stellar SDK | `internal/stellar` | 18 | Contract layer **stubbed** |
| AI service | `apps/intelligence` | 39 | Python/Prometheus |
| Mobile (Flutter) | `mobile/` | ~150 | Dart, multi-platform scaffold |
| Mobile (Expo) | `apps/mobile` | 9 | **Competing** RN/TS app |

Test ratio: Go 109 test / 181 src (good). TS 17 test / 211 src (thin). Website + both mobile apps: ~0.

---

# Findings by category

Severity: **Critical · High · Medium · Low · Trivial**. Each finding: root cause → consequence → fix → effort → risk-of-change.

## A. Architecture

### A1 — Duplicate/competing Go API: `services/api` vs `apps/api` · **High**
- **Files/modules:** `services/api/**` (18 files) vs `apps/api/**` (378).
- **Root cause:** a campaign PR scaffolded a second "production-grade" API (`services/api`) in parallel instead of extending `apps/api`. `services/api/README.md` advertises vault/user/offramp domains, but the actual tree only implements `intelligence` + a prometheus relay. `OSS_CLEANUP.md:195` records the same "parallel logic instead of fixing existing struct" pattern.
- **Consequence:** ambiguous source of truth, doubled maintenance, drift between two `domain/intelligence` + `pkg/response` copies, onboarding confusion.
- **Fix:** decide the canonical API (evidence says `apps/api`). Fold the one useful piece of `services/api` (the prometheus relay, if not already in `apps/api`) into `apps/api`, then delete `services/api`. If `services/api` is intentionally the future skeleton, do the reverse and freeze `apps/api` — but pick one.
- **Effort:** M (0.5–1 day incl. import/route reconciliation). **Risk:** Medium (deletion — needs confirmation the relay isn't wired anywhere).

### A2 — Inconsistent Go module paths across 3 `go.mod` · **High**
- **Files:** `go.mod` → `module github.com/Damola09/nester`; `apps/api/go.mod` → `github.com/suncrestlabs/nester/apps/api`; `services/api/go.mod` → `github.com/Suncrest-Labs/nester`.
- **Root cause:** modules were initialized under three different namespaces — a **contributor's personal fork** (`Damola09`), a **lowercase** org (`suncrestlabs`), and the **canonical** org (`Suncrest-Labs`). Go module paths are case-sensitive.
- **Consequence:** import paths won't resolve consistently; the root module under a personal namespace will break any external `go get` and signals abandoned scaffolding; case mismatch (`suncrestlabs` vs `Suncrest-Labs`) breaks cross-module imports and CI on case-sensitive filesystems.
- **Fix:** standardize on `github.com/Suncrest-Labs/nester[/...]`. Rename module directives + update all import statements. Confirm which module the root `go.mod` even serves (may be vestigial — a 4th finding candidate).
- **Effort:** M. **Risk:** Medium (touches every Go import; needs a build-verify pass).

### A3 — Duplicate/competing mobile apps: `mobile/` (Flutter) vs `apps/mobile` (Expo) · **Medium**
- **Files:** `mobile/lib/main.dart` + full Flutter multiplatform scaffold vs `apps/mobile/app/*.tsx` (Expo Router, 9 files).
- **Root cause:** two separate campaign efforts targeted "mobile" with different frameworks.
- **Consequence:** dead weight, split direction, ~150 Flutter files (ios/macos/android/windows/linux/web scaffolds) carried with near-zero logic.
- **Fix:** choose one mobile stack; archive/delete the other. Neither has tests, so pick by product intent.
- **Effort:** S (decision) + S (deletion). **Risk:** Low.

### A4 — Root `go.mod` under personal namespace, unclear purpose · **Medium**
- **File:** `go.mod` (`github.com/Damola09/nester`), `internal/stellar` likely its only member.
- **Root cause:** repo-root Go module left from early scaffolding; `apps/api` and `services/api` are their own modules.
- **Fix:** fold `internal/stellar` into the canonical module or make the root module the workspace root with a `go.work`; rename off the personal namespace regardless.
- **Effort:** S–M. **Risk:** Medium (import churn).

## B. Bugs / Incomplete features

### B1 — `internal/stellar` Soroban contract layer is stubbed · **High**
- **Files:** `internal/stellar/contract.go:45,52,110,211,214`, `vault_reader.go:47` (7 non-test placeholder sites).
- **Root cause:** contract invocation/simulation was never implemented — returns hardcoded `GasEstimate = 10000`, `TransactionHash = "0x00"`, `nil` invocations. Tests explicitly assert placeholder behavior (`contract_test.go:315,780`).
- **Consequence:** any feature depending on on-chain vault reads/tx submission is non-functional; silent wrong values instead of errors.
- **Fix:** implement real Soroban RPC simulate/submit, or gate the stub behind an explicit `ErrNotImplemented` + feature flag so callers fail loud. Track as a feature, not a quick fix.
- **Effort:** L. **Risk:** High (core protocol path). *Manual-only.*

### B2 — WebSocket notification channel disabled by TODO · **Medium**
- **File:** `apps/api/cmd/api/main.go:322` — `// notifications.NewWebSocketChannel(wsHub) // TODO: Fix interface implementation`.
- **Root cause:** interface mismatch left the WS notification channel commented out.
- **Consequence:** real-time notifications silently unavailable; `useWebSocket.ts` frontend hook may connect to nothing.
- **Fix:** reconcile the channel interface and re-enable, or remove the dead wiring + frontend hook if WS is out of scope.
- **Effort:** M. **Risk:** Medium.

### B3 — `seed.sql` schema drift + healthcheck path mismatch · **High** *(tracked in OSS_CLEANUP.md)*
- **Files:** `scripts/seed.sql`, `docker-compose.yml` (`/healthz` vs README `/health`).
- Already in `OSS_CLEANUP.md` blocking list (migration 007 renamed `name`→`display_name`, added `wallet_address`/`kyc_status`; seed still references `email`/`name`). Listed here for completeness. **Fix + effort:** S. **Risk:** Low. Auto-fixable.

## C. Security

### C1 — CORS / auth handling · **Disregard (verified safe)** — see Disregards.

### C2 — `.venv` present on disk but untracked · **Trivial**
- `apps/intelligence/.venv` exists locally, correctly gitignored. No action; noted to explain why intelligence "stub" grep hits were vendored-lib noise, not app code.

### C3 — `.env.example` documents a weak default JWT secret · **Low**
- **File:** `.env.example` — `AUTH_JWT_SECRET=dev-nester-jwt-secret-change-in-production`.
- **Root cause:** example file ships a memorable default. It is explicitly labeled "DO NOT use this default in production" and `.env` is gitignored, so this is acceptable **for an example**. Risk is only if a deploy copies it verbatim.
- **Fix (optional):** leave blank in `.env.example` and fail startup when `APP_ENV=production` and the secret equals the dev default. Config validation already rejects wildcard CORS in prod — mirror that.
- **Effort:** S. **Risk:** Low.

> No hardcoded live secrets found in tracked source (gitleaks config present; scan clean).

## D. Code Quality

### D1 — Committed 30 MB binary `apps/api/api.exe` · **High**
- **File:** `apps/api/api.exe` (30,872 KB, tracked).
- **Root cause:** a compiled Windows binary was committed. Bloats clone/history, platform-specific, useless artifact.
- **Fix:** `git rm --cached apps/api/api.exe`, add `*.exe` to `.gitignore`. (History rewrite optional; at minimum stop tracking.)
- **Effort:** S. **Risk:** Low. Auto-fixable.

### D2 — Committed CI scratch output `lint_out.txt`, `pr_checks.txt` · **Low**
- **Files:** root `lint_out.txt`, `pr_checks.txt`.
- **Fix:** delete + gitignore. **Effort:** S. **Risk:** none. Auto-fixable.

### D3 — `as any` casts on zod resolvers in vault modals · **Low**
- **File:** `apps/dapp/frontend/components/vault-action-modals.tsx:160,612,1006` (each with an `eslint-disable`).
- **Root cause:** `zodResolver(schema as any)` — type friction between zod + RHF versions.
- **Fix:** type the resolver generic properly (`zodResolver<FormValues>`), or upgrade `@hookform/resolvers`. Low value; a known ecosystem papercut.
- **Effort:** S. **Risk:** Low.

### D4 — Leftover per-issue context files gitignored · **Trivial**
- **File:** `.gitignore` carries `ISSUE_CONTEXT_156.md`, `_236`, `_248`, `_264`, `.solve-issues-state.json`, `.issue-worktrees/` — workflow cruft from the campaign tooling.
- **Fix:** prune the stale entries. **Effort:** S. **Risk:** none.

## E. Testing

### E1 — Frontend test coverage thin & lopsided · **Medium**
- 17 TS tests, all under `apps/dapp/frontend/__tests__`; `apps/website`, `apps/mobile`, `mobile/` have none.
- **Fix:** add smoke/render tests for critical dapp flows (offramp, vault modals, tx history — the branch under work) and at least build-level checks for website/mobile.
- **Effort:** M–L. **Risk:** Low.

> **Test taxonomy for the roadmap** — don't just "add tests"; classify the gap:
> **regression** (lock current behavior before refactors: transaction/tx-history, offramp) ·
> **integration** (`apps/api` DB + handler wiring — some exist under `db_integration_test.go`) ·
> **contract** (Soroban/`packages/contracts` ↔ `internal/stellar` boundary — currently only stub-asserting) ·
> **e2e** (deposit→yield→offramp user flow — none) ·
> **performance** (N+1 / query hotspots in portfolio/tvl — none). Prioritize regression + contract first.

### E2 — Go domain packages with zero tests · **Medium**
- ~24 `apps/api/internal/domain/*` packages have source but no `_test.go` (transaction, portfolio, user, webhook, bankaccount, yieldharvest, …). Overall Go ratio is healthy (109/181) but concentrated in infra, not domain logic.
- **Fix:** prioritize tests for money-touching domains (transaction, portfolio, bankaccount, yieldharvest, savingsschedule).
- **Effort:** L. **Risk:** Low.

### E3 — Tests assert stub behavior as correct · **Medium**
- `internal/stellar/*_test.go` lock in placeholder returns (e.g. `TestSubmitTransaction_ReturnsPlaceholderSuccess`). These will mask B1 when it's implemented.
- **Fix:** convert to skipped/pending tests or `t.Skip("pending real Soroban impl")` so they don't cement the stub.
- **Effort:** S. **Risk:** Low.

## F. Dependencies / DX

### F1 — Mixed package managers: pnpm + npm lockfiles coexist · **Medium**
- **Files:** root `pnpm-lock.yaml` (825 KB) **and** `package-lock.json`; plus `apps/dapp/frontend/package-lock.json` (584 KB). `pnpm-workspace.yaml` + `turbo.json` indicate pnpm is intended.
- **Root cause:** contributors ran `npm install` in a pnpm workspace.
- **Consequence:** divergent dependency trees, nondeterministic installs, CI/lockfile drift.
- **Fix:** standardize on pnpm; delete stray `package-lock.json` files; add a `preinstall` "only-allow pnpm" guard.
- **Effort:** S–M. **Risk:** Low–Medium (verify installs after).

### F2 — Documentation drift · **Low**
- `services/api/README.md` describes domains it doesn't implement; healthcheck path disagreement (B3); multiple overlapping audit docs at root (`OSS_CLEANUP.md`, `CI_CD_VALIDATION_REPORT.md`, `AUDIT_THREAT_MODEL.md`, `SEP24_DECISION.md`, now this).
- **Fix:** reconcile READMEs to actual state after A1/A3 decisions; consider a `docs/` folder to consolidate root-level reports.
- **Effort:** S. **Risk:** none.

## G. Contributor Consistency (post-campaign drift)

Where long-term maintenance cost hides after many contributors merge scoped work. Seeded from
evidence already gathered; marked *investigate* where a full pass is still owed.

### G1 — Duplicated HTTP response + error layers · **Medium**
- Two `pkg/response/response.go` (`apps/api`, `services/api`) and an `apps/api/pkg/apperror` — parallel response/error conventions across the two APIs. Collapses naturally once A1 picks a canonical API.
- **Fix:** single response/error package post-A1. **Effort:** S (after A1). **Risk:** Low.

### G2 — Duplicated `domain/intelligence` models · **Medium**
- `apps/api/internal/domain/intelligence` and `services/api/internal/domain/intelligence` both model the same concept. **Fix:** dedupe during A1/A9. **Risk:** Low.

### G3 — Inconsistent module namespaces · **High** *(= A2)*
- Three org spellings (`Damola09`, `suncrestlabs`, `Suncrest-Labs`) is itself contributor drift. Tracked under A2/T-08.

### G4 — Type/DTO + validation duplication (frontend) · **Low** *(investigate)*
- `Transaction` type already re-aligned once this branch (`fix/661-txhistory`) to match `TransactionTable`; `as any` on zod resolvers repeated 3× in `vault-action-modals.tsx`. Suggests shared DTO/validation modules aren't consistently used.
- **Fix:** centralize shared DTOs + zod schemas; full pass owed. **Effort:** M. **Risk:** Low.

### G5 — Logging / error-wrapping / API-response-shape consistency · **Medium** *(investigate)*
- Not yet fully scanned (scan deferred). Owed: one pass classifying log libs, `fmt.Errorf %w` vs `errors.New` vs `apperror`, and JSON response envelope shape across handlers. **Effort:** M. **Risk:** Low.

---

# Disregards (intentional — do not change)

- **CORS handling** — `apps/api` rejects `"*"` explicitly (`config.go:385`), requires `ALLOWED_ORIGINS` in production (`validateAllowedOrigins`), and has thorough tests (`server_test.go:380–528`). Correct and well-guarded.
- **`dangerouslySetInnerHTML`** in `layout.tsx`/`page.tsx`/`ai-layer.tsx` — used only for JSON-LD structured data and a theme-init script with static/`JSON.stringify`'d content. No user input. Standard Next.js pattern.
- **`_ = json.NewEncoder(w).Encode(...)`** ignored-error pattern in `pkg/response` — writing to an already-committed HTTP response; the error is unactionable. Acceptable Go idiom.
- **SQL string concatenation** hits are test-only schema DDL (`db_integration_test.go` `CREATE/DROP SCHEMA "+schema`) with `//nolint:errcheck` — not user-facing, not injectable. Safe.
- **`.env.example` JWT default** — acceptable for an example file (see C3); only worth the optional prod-guard hardening.
- **Go 1.25 directive** — current, not outdated.

---

# Cleanup Roadmap (staged — revised per maintainer strategy)

Sequencing separates **repository hygiene** from **dependency/build hygiene**, inserts a build-verify
baseline gate, and treats architecture as human RFC decisions rather than autonomous code changes.

| Stage | Theme | Tasks | Autonomous? |
|---|---|---|---|
| **0 — Baseline** | Preserve rollback point: tag audited HEAD, record CI/build status, save this report | T-00 | ✅ |
| **1 — Repo hygiene** | Zero-risk, individual atomic commits | T-01, T-02, T-03 | ✅ |
| **2 — Build-verify gate** | Run every project as-is; record known-good/failing/warnings baseline | **T-03.5** | ✅ |
| **3 — Dependency hygiene** | Audit npm usage (CI/Docker/docs) *first*, then standardize pnpm + verify | T-04 | ⚠️ verify each step |
| **4 — Production correctness** | Low-risk, high-value fixes | T-05, T-06, T-07 | ⚠️ |
| **🛑 5 — Architecture RFCs** | **HARD PAUSE — maintainer decisions, not cleanup** | A1, A3, A4 → RFCs | ❌ human |
| **6 — Module standardization** | Document every module/edge/import *first*, migrate once | T-08 | ❌ |
| **7 — Consolidation** | Delete duplicates **only after** module cleanup (preserves recoverable code) | T-09, T-11, T-10 | ❌ |
| **8 — Core feature (elevated)** | **B1 Soroban is the #1 engineering task** — not cleanup, a missing core feature | T-12, T-13 | ❌ |
| **9 — Testing** | By taxonomy: regression → contract → integration → e2e → perf | T-14…T-16 | ❌ |
| **10 — Documentation** | Last — docs describe reality after everything settles | T-18, T-17 | ❌ |

**Autonomous runway:** Stages 0→4 (T-00 through T-07). **Then stop** for architecture RFCs. Everything
after the pause is maintainer-gated. Rationale: hygiene is reversible and unblocks clean diffs;
build-verify establishes "known good" before dependency surgery; module paths get canonicalized
*before* deleting `services/api`/duplicate mobile so imports resolve and code stays recoverable;
B1 (fake Soroban layer) is elevated above generic cleanup as the highest-impact work once stable.

---

# Master TODO

Ordered so each task enables the next. Status: all **Not Started**.

| ID | Title | Category | Priority | Depends on | Complexity | Est. | Auto? |
|---|---|---|---|---|---|---|---|
| T-00 | **Baseline:** tag audited HEAD (`audit-baseline-2026-07-04`), record build/CI status, save this report | Baseline | High | — | XS | 10m | ✅ |
| T-01 | `git rm --cached apps/api/api.exe`; add `*.exe` to `.gitignore` | Quality | High | T-00 | XS | 10m | ✅ |
| T-02 | Delete `lint_out.txt`, `pr_checks.txt`; gitignore them | Quality | Low | — | XS | 5m | ✅ |
| T-03 | Prune stale `ISSUE_CONTEXT_*` / `.solve-issues-state.json` from `.gitignore` | DX | Trivial | T-00 | XS | 5m | ✅ |
| T-03.5 | **Build-verify gate:** run Go build+test, pnpm install, turbo build, frontend+Rust tests, Python checks; record pass/fail/warn baseline | Baseline | High | T-01…T-03 | S | 1h | ✅ |
| T-04 | Standardize on pnpm — **first** audit npm usage in CI/Dockerfiles/docs/Actions and fix those, then remove stray `package-lock.json` (root + frontend), enforce pnpm, verify install+build | Deps | Medium | T-03.5 | S–M | 2–3h | ⚠️ verify |
| T-05 | Fix `scripts/seed.sql` to post-007 schema (`display_name`, `wallet_address`, `kyc_status`) | Bug | High | — | S | 1h | ✅ |
| T-06 | Reconcile healthcheck path `/healthz` vs `/health` (compose ↔ README ↔ code) | Bug | High | — | XS | 20m | ✅ |
| T-07 | Add prod-startup guard rejecting default `AUTH_JWT_SECRET`; blank it in `.env.example` | Security | Low | — | S | 45m | ⚠️ |
| T-08 | Standardize all Go module paths to `github.com/Suncrest-Labs/nester[/...]`; update imports; build-verify | Arch | High | — | M | 3–4h | ❌ |
| T-09 | Decide canonical API; fold `services/api` prometheus relay into `apps/api`; delete `services/api` | Arch | High | T-08 | M | 4h | ❌ |
| T-10 | Resolve root `go.mod` namespace/purpose (fold `internal/stellar` or adopt `go.work`) | Arch | Medium | T-08 | S–M | 2h | ❌ |
| T-11 | Decide mobile stack (Flutter vs Expo); archive/delete the loser | Arch | Medium | — | S | 1h + decision | ❌ |
| T-12 | Implement real Soroban simulate/submit in `internal/stellar`, or gate stub behind `ErrNotImplemented` + flag | Bug/Feature | High | T-08 | L | 1–3d | ❌ |
| T-13 | Re-enable or remove WS notification channel (`main.go:322`) + frontend hook | Bug | Medium | — | M | 3h | ❌ |
| T-14 | Convert `internal/stellar` stub-asserting tests to `t.Skip` pending B1 | Testing | Medium | T-12 | S | 45m | ✅ |
| T-15 | Add Go tests for money-domains (transaction, portfolio, bankaccount, yieldharvest, savingsschedule) | Testing | Medium | — | L | 1–2d | ❌ |
| T-16 | Add frontend tests for offramp / vault modals / tx-history | Testing | Medium | — | M | 1d | ❌ |
| T-17 | Type zod resolvers properly; drop `as any` + eslint-disables in `vault-action-modals.tsx` | Quality | Low | T-04 | S | 1h | ⚠️ |
| T-18 | Reconcile READMEs to actual state; consider consolidating root audit docs into `docs/` | Docs | Low | T-09,T-11 | S | 1–2h | ❌ |

**Auto? legend:** ✅ safe to auto-fix + atomic-commit · ⚠️ auto with post-verify · ❌ manual (design/deletion/build-wide).

---

## Execution status

Working the autonomous runway (Stages 0→4, T-00→T-07), each an atomic commit, hard-stopping
before Stage 5 architecture RFCs.

- [ ] T-00 baseline tag + build/CI status recorded
- [ ] T-01 remove `api.exe`
- [ ] T-02 remove `lint_out.txt` / `pr_checks.txt`
- [ ] T-03 prune `.gitignore`
- [ ] T-03.5 build-verify baseline
- [ ] T-04 pnpm standardization (npm-usage audit first)
- [ ] T-05 seed.sql schema
- [ ] T-06 healthcheck path
- [ ] T-07 JWT prod guard
- [ ] **🛑 PAUSE** — A1/A3/A4 RFCs for maintainer

B1 (Soroban) elevated to highest-impact engineering task, scheduled post-pause once repo is stable.
