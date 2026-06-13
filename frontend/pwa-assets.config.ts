import { defineConfig, minimal2023Preset } from '@vite-pwa/assets-generator/config'

// Regenerates the full PWA icon set (192/512/maskable + 180 apple-touch-icon
// + favicon.ico) from a single square source image.
// Run via: `yarn pwa:generate`
export default defineConfig({
  // favicon.png is the 803×803 square brand mark already used as the favicon.
  images: ['public/favicon.png'],
  preset: minimal2023Preset,
})
