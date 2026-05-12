import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  use: {
    baseURL: 'http://localhost:4000',
    headless: true,
  },
  webServer: {
    command: 'go run .',
    url: 'http://localhost:4000/api/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
