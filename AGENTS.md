# go-pixo Development Guide

Go → WASM PNG/JPEG compression library with web UI (client-side only).

## Project Snapshot

Simple monorepo with:
- **Go backend** (PNG/JPEG encoders + WASM bridge)
- **Web frontend** (React + Rescript + Vite)
- **Documentation** (Learning materials in `docs/`)

Each package has its own AGENTS.md - see JIT Index below.

## Root Setup Commands

```bash
# Install Go dependencies
go mod download

# Install web dependencies  
cd web && npm install

# Build WASM bridge
./scripts/build-wasm.sh

# Run web dev server
cd web && npm run dev

# Run all tests
go test ./... && cd web && npm test
```

## Universal Conventions

**Go Code Style**:
- Imports: std lib first, then local (full module path)
- Naming: Exported = PascalCase, private = camelCase
- Error handling: Return `error` as second value, never suppress
- Comments: Godoc on exported functions

**Web Code Style**:
- Rescript + React functional components
- Tailwind CSS for styling
- TypeScript for type safety
- No warnings allowed

**Testing**:
- Go: Table-driven tests with `t.Run`
- Web: Vitest for unit, Playwright for E2E

## Security & Secrets

- Never commit tokens, API keys, or credentials
- Use `.env` files (already in `.gitignore`)
- WASM compilation requires Go 1.25+ and wasm_exec.js

## JIT Index (what to open, not what to paste)

### Package Structure
- Core compression: `src/compress/` → [see src/compress/AGENTS.md](src/compress/AGENTS.md)
- PNG encoder: `src/png/` → [see src/png/AGENTS.md](src/png/AGENTS.md)
- JPEG encoder: `src/jpeg/` → [see src/jpeg/AGENTS.md](src/jpeg/AGENTS.md)
- WASM bridge: `src/wasm/` → [see src/wasm/AGENTS.md](src/wasm/AGENTS.md)
- Web UI: `web/` → [see web/AGENTS.md](web/AGENTS.md)

### Quick Find Commands
```bash
# Find Go function
rg -n "func Name" src/**/*.go

# Find Go test
rg -n "func Test" src/**/*_test.go

# Find React component
rg -n "export.*Component" web/src/**/*.res

# Find API/WASM bridge
rg -n "//go:build js" src/wasm/*.go
```

## Working Rules (keep delivery fast)

These apply to every agent and delegate working in this repo.

**Roles**
- The orchestrator decides, briefs, checks evidence, and opens and merges PRs.
- Planning and implementation run on a strong coding model. Review runs on an independent model.
- At most 3 delegates run at once.

**Tests: keep them proportionate**
- For each behaviour you add or fix, write one focused test of the user story, plus one regression test for the bug.
- Don't build matrices, throwaway probe batteries, or repeated fail-before proofs. One proof is enough.
- Add Playwright e2e only when the user-visible flow changes.

**Checks: run the full set once, on the final head**
- Go: `go test ./...` and golangci-lint v2 on both targets.
- Web: `bun run build`, `bun run typecheck`, `bun run test`.
- E2E: `bun run test:e2e` and `bunx playwright test --config playwright.offline.config.ts`, only when the web path changed.
- The reviewer runs only the relevant tests and reads the executor's results.

**Review gating**
- Only findings of medium severity or higher block a merge, meaning a real user story breaks or a security property fails.
- Collect low findings and nits in one follow-up GitHub issue, then merge the PR.
- Small follow-ups (text only, or a few lines with no behaviour change) skip another review round.

**Branching and merging**
- Run `git fetch origin`, then branch from `origin/master`. T3 worktrees can start from a stale HEAD.
- Merge one PR at a time, with a merge commit: `gh pr merge <n> --merge --match-head-commit <sha>`. Merge only after the required checks pass.
- After a merge, run `git pull --ff-only` in the main checkout, then remove the merged worktree.

**Issues and changelog**
- Open a GitHub issue before work starts when practical.
- Each PR body gets one `Closes #N` line per issue, and a `## Changelog` section with user-visible bullets.

**Benchmarks**
- When performance or compression changes, compare the before and after production builds with `web/e2e/benchmark/compare-versions.mjs`.
- Record the results under `docs/benchmarks/`.

**Files**
- Keep UTF-8 with LF endings.
- On Windows, never write files with PowerShell `Set-Content` or `Out-File` without `-Encoding utf8`.

**Shared devices**
- The Android emulators on the maintainer's PC belong to another project.
- Before using one, take `C:\DevTools\locks\flutter-ui-payment-device.lock`: create it with `New-Item -ItemType Directory`, and retry every 60 s if it exists.
- Use one emulator at a time, and restore any setting you change.
- Release the lock when you're done, including on failure.

## Definition of Done

Before merging (run once on the final head):
- [ ] `go test ./...` passes
- [ ] `golangci-lint run ./...` and `GOOS=js GOARCH=wasm golangci-lint run ./...` pass (golangci-lint v2)
- [ ] In `web/`: `bun run build`, `bun run typecheck`, `bun run test` pass
- [ ] Playwright e2e (default and offline configs) pass when the web path changed
- [ ] Required CI checks pass on the PR head
