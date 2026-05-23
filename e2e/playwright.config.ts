import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  workers: 1,
  use: {
    baseURL: 'http://localhost:4000',
    headless: true,
  },
  webServer: {
    command: process.env.CI ? './server' : 'go run .',
    url: 'http://localhost:4000/api/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
