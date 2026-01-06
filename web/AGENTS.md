# Web UI Package

React + Rescript + Vite frontend for image compression (client-side).

## Package Identity

- **Purpose**: User interface for PNG/JPEG compression via WASM
- **Technology**: React 19, Rescript, TypeScript, Tailwind CSS, Vite
- **Testing**: Vitest (unit), Playwright (E2E)

## Setup & Run

```bash
# Install dependencies
npm install

# Dev server (with hot reload)
npm run dev

# Build for production
npm run build

# Preview build
npm run preview

# Run tests
npm test

# Run E2E tests
npm run test:e2e

# Typecheck
npm run rescript:build
```

## Patterns & Conventions

**File Organization**:
```
web/src/
├── App.res              # Main component
├── components/          # ReScript React components
├── worker.ts           # Web Worker (WASM bridge)
└── types.res           # TypeScript types

web/e2e/
├── conversion.spec.ts   # Playwright E2E tests
└── fixtures/           # Test fixtures
```

**Key Patterns**:
- ✅ DO: Use Rescript functional components (`@react.component`)
- ✅ DO: Use Tailwind classes (never inline styles)
- ✅ DO: Keep state in React hooks (`useReducer` for complex state)
- ✅ DO: Use Web Worker for WASM (off main thread)
- ❌ DON'T: Mix Rescript and TypeScript in same file
- ❌ DON'T: Block main thread with synchronous operations

**Component Pattern** (see `web/src/components/FileQueue.res:23-67`):
```rescript
@react.component
let make = (~prop: type, ()) => {
  let (state, dispatch) = React.useReducer(reducer, initialState)
  
  <div className="tailwind-class">
    {React.string("Content")}
  </div>
}
```

**State Management** (see `web/src/App.res:55-166`):
- ✅ DO: Use single `useReducer` for app state
- ✅ DO: Use action types for updates
- ❌ DON'T: Use multiple `useState` for related state

**WASM Integration** (see `web/src/worker.ts:125-179`):
- ✅ DO: Initialize WASM once in worker
- ✅ DO: Use `postMessage` for async communication
- ✅ DO: Handle progress callbacks
- ❌ DON'T: Call WASM directly from main thread

## Touch Points / Key Files

- **Main App**: `web/src/App.res:168-631` (state + routing)
- **Bottom Bar**: `web/src/components/BottomBar.res:23-203` (controls)
- **Worker**: `web/src/worker.ts:1-392` (WASM bridge)
- **Types**: `web/src/types.res:61-75` (app state)

## JIT Index Hints

```bash
# Find Rescript components
rg -n "@react.component" web/src/**/*.res

# Find state actions
rg -n "type action" web/src/**/*.res

# Find WASM calls
rg -n "encodePng\|encodeJpeg" web/src/**/*.ts

# Find E2E tests
ls web/e2e/*.spec.ts
```

## Common Gotchas

- **Rescript Syntax**: `.res` files (not `.tsx`), camelCase for variables
- **Type Conversion**: Rescript ↔ TypeScript boundary requires care
- **WASM Path**: Built WASM must be in `web/public/main.wasm`
- **Bundle Size**: Vite builds can grow large (use code splitting)
- **Browser Support**: WASM requires modern browsers (ES2017+)

## Pre-PR Checks

```bash
npm run build && npm test && npm run test:e2e
```
