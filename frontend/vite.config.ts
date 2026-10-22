import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
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
  server: {
    port: 3000,
  },
  define: {
    __DATA_MODE__: JSON.stringify(process.env.VITE_DATA_MODE || 'mock'),
    __API_BASE_URL__: JSON.stringify(process.env.VITE_API_BASE_URL || 'http://localhost:8080'),
    __SSE_ENDPOINT__: JSON.stringify(process.env.VITE_SSE_ENDPOINT || 'http://localhost:8080/api/vehicles/updates'),
  },
});
