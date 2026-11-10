import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react-swc';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{js,jsx,ts,tsx}'],
    // Default 5s has no headroom on this suite's profile: ~205 jsdom environments
    // with heavy module graphs saturate all workers, and purely synchronous
    // render tests (e.g. EmployeeAttendanceActionDock: 679ms isolated) cross 5s
    // of wall time while workers contend for cores (PAYR-35). Assertions are
    // unaffected — this only widens the starvation allowance.
    testTimeout: 15000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
