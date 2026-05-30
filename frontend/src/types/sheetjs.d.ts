// Ambient typings for SheetJS CDN ESM module (Community 0.20.2)
// This provides the minimal surface used in our codebase while avoiding the vulnerable npm package.

declare module 'https://cdn.sheetjs.com/xlsx-0.20.2/package/xlsx.mjs' {
  export interface WorkSheet {
    [key: string]: unknown;
    '!ref'?: string;
    '!cols'?: Array<{ wch?: number }>;
  }

  export interface WorkBook {
    SheetNames: string[];
    Sheets: Record<string, WorkSheet>;
    [key: string]: unknown;
  }

  export const read: (
    data: ArrayBuffer | Uint8Array,
    opts?: { type?: 'array' | 'buffer' | 'binary' | string }
  ) => WorkBook;

  export const writeFile: (
    wb: WorkBook,
    filename: string,
    opts?: { bookType?: 'xls' | 'xlsx' | string }
  ) => void;

  export const utils: {
    sheet_to_json: (
      ws: WorkSheet,
      opts?: { header?: 1 | string[] | unknown[] }
    ) => unknown[] | unknown[][];
    aoa_to_sheet: (data: unknown[][]) => WorkSheet;
    json_to_sheet: (
      data: Record<string, unknown>[],
      opts?: { header?: string[] }
    ) => WorkSheet;
    book_new: () => WorkBook;
    book_append_sheet: (wb: WorkBook, ws: WorkSheet, name: string) => void;
  };
}

