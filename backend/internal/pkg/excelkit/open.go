package excelkit

import (
	"bytes"
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

// OpenWorkbook opens a workbook from a reader, converting a legacy BIFF .xls
// (OLE2 compound document) into an in-memory xlsx so excelize-only parsers
// keep working unchanged. Detection is by magic bytes, not filename, so a
// renamed file cannot take the wrong path — wrong content still fails on
// open. Callers bound the input (uploadguard caps uploads at MaxBytes);
// the .xls path re-checks against the unzip cap before parsing.
func OpenWorkbook(r io.Reader) (*excelize.File, error) {
	head := make([]byte, ole2SignatureLen)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	head = head[:n]
	if isOLE2(head) {
		return openXLS(io.MultiReader(bytes.NewReader(head), r))
	}
	return OpenReader(io.MultiReader(bytes.NewReader(head), r))
}

const ole2SignatureLen = 8

// isOLE2 reports whether head starts with the OLE2 compound document
// signature (legacy .xls).
func isOLE2(head []byte) bool {
	ole2 := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	if len(head) < len(ole2) {
		return false
	}
	for i, b := range ole2 {
		if head[i] != b {
			return false
		}
	}
	return true
}

// OpenFile opens a workbook from a path with decompression caps applied.
func OpenFile(path string) (*excelize.File, error) {
	return excelize.OpenFile(path, excelize.Options{
		UnzipSizeLimit:    UnzipSizeLimit(),
		UnzipXMLSizeLimit: UnzipXMLSizeLimit(),
	})
}
