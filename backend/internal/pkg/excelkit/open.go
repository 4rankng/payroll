package excelkit

import (
	"io"
	"os"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// Decompression caps. excelize defaults allow ~16 GiB of unzip output, which
// turns a crafted upload into an OOM vector; every reader-side workbook open
// in this codebase should go through OpenReader/OpenFile instead of raw
// excelize. Defaults bound a decompression bomb while still covering dense
// legitimate workbooks — a ~12,000-row employee import or a re-uploaded
// sao-kê export decompresses past the old 50/10 MiB limits.
const (
	// DefaultUnzipSizeLimit caps total unzip output.
	DefaultUnzipSizeLimit = 64 << 20
	// DefaultUnzipXMLSizeLimit caps each individual XML stream in the zip.
	DefaultUnzipXMLSizeLimit = 32 << 20
)

// UnzipSizeLimit returns the effective total-unzip cap. EXCEL_UNZIP_MAX_BYTES
// overrides the default (parsed once per call; invalid values fall back
// silently), mirroring uploadguard's EXCEL_UPLOAD_MAX_BYTES escape hatch.
func UnzipSizeLimit() int64 {
	if v, err := strconv.ParseInt(os.Getenv("EXCEL_UNZIP_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		return v
	}
	return DefaultUnzipSizeLimit
}

// UnzipXMLSizeLimit returns the effective per-XML-stream cap.
// EXCEL_XML_MAX_BYTES overrides the default (invalid values fall back
// silently).
func UnzipXMLSizeLimit() int64 {
	if v, err := strconv.ParseInt(os.Getenv("EXCEL_XML_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		return v
	}
	return DefaultUnzipXMLSizeLimit
}

// OpenReader opens a workbook from a reader with decompression caps applied.
func OpenReader(r io.Reader) (*excelize.File, error) {
	return excelize.OpenReader(r, excelize.Options{
		UnzipSizeLimit:    UnzipSizeLimit(),
		UnzipXMLSizeLimit: UnzipXMLSizeLimit(),
	})
}

// OpenFile opens a workbook from a path with decompression caps applied.
func OpenFile(path string) (*excelize.File, error) {
	return excelize.OpenFile(path, excelize.Options{
		UnzipSizeLimit:    UnzipSizeLimit(),
		UnzipXMLSizeLimit: UnzipXMLSizeLimit(),
	})
}
