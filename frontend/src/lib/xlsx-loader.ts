// Lightweight loader for SheetJS CE 0.20.2 via official CDN.
// Memoizes the dynamic import to avoid multiple network requests.

export type XLSXModule = typeof import('https://cdn.sheetjs.com/xlsx-0.20.2/package/xlsx.mjs');

let xlsxPromise: Promise<XLSXModule> | null = null;

export async function getXLSX(): Promise<XLSXModule> {
  if (!xlsxPromise) {
    xlsxPromise = import('https://cdn.sheetjs.com/xlsx-0.20.2/package/xlsx.mjs');
  }
  return xlsxPromise;
}

