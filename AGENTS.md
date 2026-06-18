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
cd web && bun install

# Build WASM bridge
./scripts/build-wasm.sh

# Run web dev server
cd web && bun run dev

# Run all tests
go test ./... && cd web && bun test --run
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

## Definition of Done

Before committing:
- [ ] `go test ./...` passes
- [ ] `golangci-lint run` passes (Go)
- [ ] `cd web && bun run build` passes (Web)
- [ ] All tests green (Go + Web)
