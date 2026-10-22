import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/tests/setup.ts'],
    css: true,
    testTimeout: 10000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'json-summary'],
      include: ['src/**/*.ts', 'src/**/*.tsx'],
      exclude: [
        '**/*.test.ts',
        '**/*.test.tsx',
        '**/node_modules/**',
        '**/dist/**',
      ],
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70,
    },
  },
  resolve: {
    alias: {
      '@config': `${__dirname}/src/config`,
      '@domain': `${__dirname}/src/domain`,
      '@application': `${__dirname}/src/application`,
      '@infrastructure': `${__dirname}/src/infrastructure`,
      '@presentation': `${__dirname}/src/presentation`,
      '@tests': `${__dirname}/src/tests`,
    },
  },
});
