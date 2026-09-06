package excelkit

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// Decompression caps, mirrored from the wallet-bulk parser (the in-repo gold
// standard). excelize defaults allow ~16 GiB of unzip output, which turns a
// crafted upload into an OOM vector; every reader-side workbook open in this
// codebase should go through OpenReader instead of excelize.OpenReader.
const (
	// DefaultUnzipSizeLimit caps total unzip output. 50 MiB comfortably
	// covers a 5,000-row workbook.
	DefaultUnzipSizeLimit = 50 << 20
	// DefaultUnzipXMLSizeLimit caps each individual XML stream in the zip.
	DefaultUnzipXMLSizeLimit = 10 << 20
)

// OpenReader opens a workbook from a reader with decompression caps applied.
func OpenReader(r io.Reader) (*excelize.File, error) {
	return excelize.OpenReader(r, excelize.Options{
		UnzipSizeLimit:    DefaultUnzipSizeLimit,
		UnzipXMLSizeLimit: DefaultUnzipXMLSizeLimit,
	})
}

// OpenFile opens a workbook from a path with decompression caps applied.
func OpenFile(path string) (*excelize.File, error) {
	return excelize.OpenFile(path, excelize.Options{
		UnzipSizeLimit:    DefaultUnzipSizeLimit,
		UnzipXMLSizeLimit: DefaultUnzipXMLSizeLimit,
	})
}
