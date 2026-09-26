// Renders the PWA icon set from public/logo-square.png (the brand logo).
// Sizes follow the manifest + index.html links; the maskable variant keeps the
// logo inside the 80% safe zone on the logo's own black background.
// favicon.ico is written as a PNG-in-ICO (32px) — valid for all modern
// browsers. Run: node scripts/render-icons.mjs
import { chromium } from "@playwright/test";
import { readFileSync, writeFileSync } from "node:fs";

const SRC = "public/logo-square.png";

const targets = [
  { file: "public/pwa-64x64.png", size: 64 },
  { file: "public/pwa-192x192.png", size: 192 },
  { file: "public/pwa-512x512.png", size: 512 },
  { file: "public/apple-touch-icon-180x180.png", size: 180 },
  { file: "public/favicon.png", size: 64 },
  { file: "public/maskable-icon-512x512.png", size: 512, pad: 0.8 },
];

const srcDataUrl =
  "data:image/png;base64," + readFileSync(SRC).toString("base64");

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 600, height: 600 } });
await page.setContent(`<!doctype html><body style="margin:0"></body>`);

// Draw the source at `size` (optionally padded onto the logo's black bg).
const renderPng = async ({ size, pad }) =>
  page.evaluate(
    async ({ size, pad, src }) => {
      const img = new Image();
      img.src = src;
      await img.decode();
      const c = document.createElement("canvas");
      c.width = size;
      c.height = size;
      const g = c.getContext("2d");
      if (pad) {
        g.fillStyle = "#000000";
        g.fillRect(0, 0, size, size);
        const inner = Math.round(size * pad);
        const off = Math.round((size - inner) / 2);
        g.drawImage(img, off, off, inner, inner);
      } else {
        g.imageSmoothingQuality = "high";
        g.drawImage(img, 0, 0, size, size);
      }
      return c.toDataURL("image/png").split(",")[1];
    },
    { size, pad, src: srcDataUrl }
  );

// PNG-in-ICO container: 6-byte header + one 16-byte directory entry + PNG.
function wrapIco(pngBuffer, size) {
  const header = Buffer.alloc(6);
  header.writeUInt16LE(0, 0); // reserved
  header.writeUInt16LE(1, 2); // type: icon
  header.writeUInt16LE(1, 4); // count
  const entry = Buffer.alloc(16);
  entry.writeUInt8(size === 256 ? 0 : size, 0); // width
  entry.writeUInt8(size === 256 ? 0 : size, 1); // height
  entry.writeUInt8(0, 2); // palette
  entry.writeUInt8(0, 3);
  entry.writeUInt16LE(1, 4); // planes
  entry.writeUInt16LE(32, 6); // bpp
  entry.writeUInt32LE(pngBuffer.length, 8);
  entry.writeUInt32LE(22, 12); // offset: 6 + 16
  return Buffer.concat([header, entry, pngBuffer]);
}

for (const t of targets) {
  const b64 = await renderPng(t);
  writeFileSync(t.file, Buffer.from(b64, "base64"));
  console.log("wrote", t.file, t.size + "px");
}

// favicon.ico at 32px
const b64 = await renderPng({ size: 32 });
writeFileSync("public/favicon.ico", wrapIco(Buffer.from(b64, "base64"), 32));
console.log("wrote public/favicon.ico 32px");

await browser.close();
