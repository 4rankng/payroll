import type { PdfMake } from 'pdfmake/build/pdfmake';

// Type definitions for pdfMake with VFS support
interface PdfMakeWithVfs extends PdfMake {
  vfs: Record<string, string>;
}

interface WindowWithPdfMake extends Window {
  pdfMake?: PdfMakeWithVfs;
}

interface GlobalThisWithPdfMake {
  pdfMake?: {
    vfs?: Record<string, string>;
  };
}

interface FontsModule {
  pdfMake?: {
    vfs?: Record<string, string>;
  };
}

// CDN URLs for fallback
const CDN_URLS = {
  pdfMake: 'https://cdnjs.cloudflare.com/ajax/libs/pdfmake/0.2.7/pdfmake.min.js',
  vfsFonts: 'https://cdnjs.cloudflare.com/ajax/libs/pdfmake/0.2.7/vfs_fonts.min.js',
};

// Load script from CDN as fallback
async function loadScriptFromCDN(url: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = url;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error(`Failed to load script from ${url}`));
    document.head.appendChild(script);
  });
}

// Try to load from CDN as fallback
async function loadPdfMakeFromCDN(): Promise<PdfMakeWithVfs> {

  try {
    // Load pdfMake from CDN
    await loadScriptFromCDN(CDN_URLS.pdfMake);

    // Check if pdfMake is available globally
    const win = window as WindowWithPdfMake;
    if (!win.pdfMake || typeof win.pdfMake.createPdf !== 'function') {
      throw new Error('pdfMake not available after CDN load');
    }


    // Load fonts from CDN
    try {
      await loadScriptFromCDN(CDN_URLS.vfsFonts);
    } catch (fontError) {
      console.warn('[pdfMake] Font loading from CDN failed:', fontError);
    }

    return win.pdfMake;
  } catch (error) {
    console.error('[pdfMake] CDN fallback failed:', error);
    throw error;
  }
}

// Ensure pdfMake is loaded before fonts to avoid race conditions.
export async function loadPdfMake(): Promise<PdfMake> {
  // Check if already loaded globally
  const win = window as WindowWithPdfMake;
  if (win.pdfMake && typeof win.pdfMake.createPdf === 'function') {
    return win.pdfMake;
  }

  try {
    const pdfMakeModule = await import('pdfmake/build/pdfmake');
    const pdfMake = (pdfMakeModule.default || pdfMakeModule) as PdfMakeWithVfs;

    if (!pdfMake || typeof pdfMake.createPdf !== 'function') {
      throw new Error('pdfMake failed to load');
    }

    // Set pdfMake globally for vfs_fonts to access (required in production builds)
    if (typeof window !== 'undefined') {
      win.pdfMake = pdfMake;
    }


    try {
      // vfs_fonts expects a global pdfMake in some builds; load after pdfmake.
      const fontsModule = (await import('pdfmake/build/vfs_fonts')) as FontsModule;

      const vfsFromModule = fontsModule?.pdfMake?.vfs;
      const vfsFromGlobal = (globalThis as GlobalThisWithPdfMake)?.pdfMake?.vfs;
      const vfs = vfsFromModule || vfsFromGlobal;

      if (vfs) {
        if (!pdfMake.vfs) {
          pdfMake.vfs = vfs;
        }
      } else {
        console.warn('[pdfMake] VFS fonts not found, PDF may not render properly');
      }
    } catch (fontError) {
      console.error('[pdfMake] Font loading failed:', fontError);
      // If fonts fail to load, leave as-is; callers may handle/report.
    }

    return pdfMake;
  } catch (error) {
    console.error('[pdfMake] Failed to load pdfMake from npm package:', error);

    // Try CDN fallback
    try {
      return await loadPdfMakeFromCDN();
    } catch (cdnError) {
      console.error('[pdfMake] CDN fallback also failed:', cdnError);
      throw new Error('Không thể tải thư viện tạo PDF. Vui lòng thử lại.');
    }
  }
}

