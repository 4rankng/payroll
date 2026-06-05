import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";
import { VitePWA } from "vite-plugin-pwa";
import type { PluginOption } from "vite";
import packageJson from "./package.json";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    // Giới hạn chỉ lắng nghe localhost và tắt CORS để tránh trang web khác đọc nội dung dev server
    host: "127.0.0.1",
    port: 3000,
    cors: false,
  },
  preview: {
    host: "127.0.0.1",
    port: 4173,
    cors: false,
  },
  define: {
    __APP_VERSION__: JSON.stringify(packageJson.version),
  },
  optimizeDeps: {
    exclude: ['chunk-SFLVWKAX', 'chunk-YV7AFGPB'],
  },
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.png', 'favicon.svg', 'robots.txt'],
      manifest: {
        name: 'TingTing',
        short_name: 'TingTing',
        description: 'TingTing',
        theme_color: '#3b82f6',
        background_color: '#ffffff',
        display: 'standalone',
        orientation: 'portrait-primary',
        icons: [
          {
            src: '/favicon.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'any maskable'
          },
          {
            src: '/favicon.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any maskable'
          }
        ]
      },
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      injectManifest: {
        maximumFileSizeToCacheInBytes: 3 * 1024 * 1024, // 3 MB for pdfMake
      },
      // Enable the service worker in `vite dev` so push notifications can
      // be tested locally. Without this, sw.ts is only emitted by the
      // production build and `navigator.serviceWorker.ready` hangs forever.
      devOptions: {
        enabled: true,
        type: 'module',
        navigateFallback: 'index.html',
      }
    }) as PluginOption
  ].filter(Boolean),
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    target: 'esnext',
    rollupOptions: {
      output: {
        // Enable filename hashing for cache busting
        entryFileNames: `assets/[name].[hash].js`,
        chunkFileNames: `assets/[name].[hash].js`,
        assetFileNames: `assets/[name].[hash].[ext]`,
        manualChunks: {
          // Vendor chunk for large libraries
          vendor: ['react', 'react-dom'],
          ui: ['@radix-ui/react-dialog', '@radix-ui/react-dropdown-menu', '@radix-ui/react-select'],
          icons: ['lucide-react'],
          utils: ['axios', '@tanstack/react-query'],
          charts: ['chart.js', 'recharts'],
          // cspell:disable-next-line
          pdfmake: ['pdfmake/build/pdfmake', 'pdfmake/build/vfs_fonts'],
        },
      },
    },
    chunkSizeWarningLimit: 1000,
    // Generate source maps for better debugging
    sourcemap: mode !== 'production',
  },
}));
