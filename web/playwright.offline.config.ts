import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  // Deliberately outside the default *.spec.ts glob used by the dev-server suite.
  testMatch: 'offline.production.ts',
  // A dev-suite run must not delete production traces/downloads from the shared default directory.
  outputDir: './test-results-offline',
  timeout: 90000,
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    serviceWorkers: 'allow',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'offline-chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    // The caller builds dist once before this suite, including ReScript and Go WASM.
    command: 'bunx vite preview --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: false,
  },
})
