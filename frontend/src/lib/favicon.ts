/**
 * Utilities to ensure the favicon/touch icon are square by
 * drawing the provided image into a square canvas and
 * swapping the <link> tags with a data URL. Kept outside TSX per repo guidelines.
 */

type FaviconOptions = {
  /** Target icon size (pixels). iOS touch icon is 180 */
  size?: number;
  /** Background fill to avoid letterboxing transparency on some platforms */
  background?: string;
};

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    // Same-origin asset; keep explicit to avoid accidental tainting
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = (e) => reject(e);
    img.src = url;
  });
}

async function rasterizeSquare(url: string, opts: FaviconOptions = {}): Promise<string> {
  const size = opts.size ?? 180;
  const bg = opts.background ?? '#ffffff';
  const img = await loadImage(url);

  const canvas = document.createElement('canvas');
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('Canvas 2D context unavailable');

  // Fill background to avoid transparent edges looking odd on iOS
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, size, size);

  // Scale to fit within the square while preserving aspect ratio
  const scale = Math.min(size / img.width, size / img.height);
  const drawW = Math.round(img.width * scale);
  const drawH = Math.round(img.height * scale);
  const dx = Math.floor((size - drawW) / 2);
  const dy = Math.floor((size - drawH) / 2);
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = 'high';
  ctx.drawImage(img, dx, dy, drawW, drawH);

  return canvas.toDataURL('image/png');
}

function upsertLink(rel: string, href: string, sizes?: string) {
  let link = document.querySelector(`link[rel="${rel}"]`) as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement('link');
    link.rel = rel;
    document.head.appendChild(link);
  }
  if (sizes) link.sizes = sizes;
  link.href = href;
}

export async function ensureSquareFavicon(imagePath: string, options?: FaviconOptions) {
  if (typeof window === 'undefined' || !document?.head) return; // SSR safe guard
  try {
    const dataUrl180 = await rasterizeSquare(imagePath, { size: 180, ...options });
    const dataUrl32 = await rasterizeSquare(imagePath, { size: 32, ...options });

    // Standard favicon (32x32)
    upsertLink('icon', dataUrl32);
    // Apple touch icon (180x180)
    upsertLink('apple-touch-icon', dataUrl180);
  } catch (err) {
    // Silently ignore to avoid blocking app start; original <link> remains
    console.warn('[favicon] Could not generate square icons:', err);
  }
}
