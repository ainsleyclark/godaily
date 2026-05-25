import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: '*.preview.spec.ts',
  workers: 1,
  timeout: 60_000,
  use: {
    baseURL: process.env.BASE_URL!,
    headless: true,
  },
});
