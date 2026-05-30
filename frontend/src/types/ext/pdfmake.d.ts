declare module 'pdfmake/build/pdfmake' {
  export interface TCreatedPdf {
    download: (defaultFileName?: string) => void;
    open: () => void;
    print: () => void;
    getBlob: (callback: (blob: Blob) => void) => void;
  }

  export interface PdfMake {
    vfs: Record<string, string>;
    createPdf: (docDefinition: unknown) => TCreatedPdf;
  }

  const pdfMake: PdfMake;
  export default pdfMake;
}

declare module 'pdfmake/build/vfs_fonts' {
  const pdfFonts: { pdfMake: { vfs: Record<string, string> } };
  export default pdfFonts;
}
