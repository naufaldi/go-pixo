# Go-Pixo Web App

React + ReScript + Tailwind v4 web application for client-side PNG compression using Go WASM.

## Setup

1. Install dependencies:
```bash
npm install
# or
bun install
```

2. Build ReScript:
```bash
npm run rescript:build
```

3. Build WASM (from repo root):
```bash
./scripts/build-wasm.sh
```

4. Run dev server:
```bash
npm run dev
```

This runs both ReScript watch mode and Vite dev server concurrently.

## Testing

### Unit Tests (Vitest)
```bash
npm test
```

### E2E Tests (Playwright)
```bash
npm run test:e2e
```

Place test images in `web/e2e/fixtures/` for conversion tests.

## Architecture

- **ReScript** (`src/*.res`) - Type-safe UI components
- **React** - Component framework
- **Tailwind CSS v4** - Styling (CSS-first, no config)
- **Go WASM** (`public/main.wasm`) - PNG compression engine
- **Vite** - Build tool and dev server

## Development

- ReScript files compile to `.res.js` (in-source)
- Import ReScript modules: `import { App } from './App.res.js'`
- WASM is loaded automatically on page load
- Tailwind classes work directly (no config needed)
