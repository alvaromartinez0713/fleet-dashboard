import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'url'
import { dirname } from 'path'

const __dirname = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [react()],
  
  // ========================================================================
  // Module Resolution: Configure path aliases for cleaner imports
  // ========================================================================
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

  // ========================================================================
  // Dev Server Configuration
  // ========================================================================
  server: {
    // host: '0.0.0.0' - Allow connections from Docker host and external clients
    // This is essential for development inside Docker containers
    host: '0.0.0.0',
    
    // port: 3000 - Standard port for frontend service
    port: 3000,
    
    // HMR (Hot Module Replacement) configuration for Docker environments
    // hmr.host: 'localhost' - Client WebSocket connects to localhost from host machine
    // hmr.port: 3000 - Same port as dev server for hot reload
    // This ensures hot reload works when dev server runs inside Docker
    hmr: {
      host: 'localhost',
      port: 3000,
    },
  },

  // ========================================================================
  // Build Configuration: Production bundle optimization
  // ========================================================================
  build: {
    // target: 'ES2020' - Use modern JavaScript features, smaller bundle size
    target: 'ES2020',
    // sourcemap: true - Generate source maps for production debugging
    sourcemap: true,
    // minify: 'terser' - Aggressive minification for smaller bundle
    minify: 'terser',
  },
})

