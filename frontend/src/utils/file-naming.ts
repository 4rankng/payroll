export function sanitizeFilename(name: string, fallback = 'download.pdf'): string {
  if (!name) return fallback;
  // Replace invalid characters and collapse whitespace/underscores
  const cleaned = name
    .replace(/[\\/:*?"<>|]/g, '_')
    .replace(/\s+/g, '_')
    .replace(/_+/g, '_')
    .trim();

  // Ensure it ends with .pdf when intended for PDF downloads
  const withExt = cleaned.toLowerCase().endsWith('.pdf') ? cleaned : `${cleaned}.pdf`;
  return withExt || fallback;
}

