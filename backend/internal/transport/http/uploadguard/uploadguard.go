// Package uploadguard validates multipart Excel uploads BEFORE any bytes
// reach a parser: request-body cap, file-size cap, and content magic sniff.
// It is the shared form of the wallet-bulk handler's proven defense-in-depth
// sequence, adopted by every Excel import endpoint.
//
// Why each layer matters:
//   - http.MaxBytesReader BEFORE c.FormFile caps the whole request body, so
//     an oversized upload is rejected while streaming in, not after.
//   - the fileHeader.Size check catches oversized files even when the
//     multipart stream lies about its length.
//   - the magic sniff rejects non-Excel content regardless of what the
//     Content-Type or extension claims (ZIP local header for .xlsx, OLE2
//     compound document for legacy .xls).
//   - excelize's own UnzipSizeLimit (see pkg/excelkit) is the last layer,
//     defending against bombs that pass all of the above.
package uploadguard

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"api-server/internal/transport/http/response"
)

// DefaultMaxBytes caps one Excel upload at 20 MiB — comfortably above every
// production workbook while bounding request memory.
const DefaultMaxBytes = 20 << 20

// MaxBytes returns the effective cap: EXCEL_UPLOAD_MAX_BYTES overrides the
// default (parsed once per call; invalid values fall back silently).
func MaxBytes() int64 {
	if v, err := strconv.ParseInt(os.Getenv("EXCEL_UPLOAD_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		return v
	}
	return DefaultMaxBytes
}

// Validate checks the "file" multipart field (body cap → size cap → magic
// sniff) and returns the file header for further use. On rejection it has
// already written the error response; the bool reports validity.
func Validate(c *gin.Context, allowXLS bool) (*multipart.FileHeader, bool) {
	limit := MaxBytes()

	// Step 1: cap the request body BEFORE c.FormFile reads it.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+512)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.BadRequest(c, "File vượt quá giới hạn dung lượng")
			return nil, false
		}
		response.BadRequest(c, "File không hợp lệ")
		return nil, false
	}

	// Step 2: declared size check.
	if fileHeader.Size > limit {
		response.BadRequest(c, "File vượt quá giới hạn dung lượng")
		return nil, false
	}

	// Step 3: open + magic sniff, then rewind for the caller.
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return nil, false
	}
	defer func() { _ = file.Close() }()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	head = head[:n]

	if !excelMagic(head, allowXLS) {
		response.BadRequest(c, "File phải là file Excel (.xlsx) hợp lệ")
		return nil, false
	}
	return fileHeader, true
}

// Receive validates (see Validate) and then reads the file with a hard cap,
// returning the bytes and the original filename. On rejection the error
// response is already written.
func Receive(c *gin.Context, allowXLS bool) (body []byte, filename string, ok bool) {
	fileHeader, ok := Validate(c, allowXLS)
	if !ok {
		return nil, "", false
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return nil, "", false
	}
	defer func() { _ = file.Close() }()

	limit := MaxBytes()
	fileBytes, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return nil, "", false
	}
	if int64(len(fileBytes)) > limit {
		// The multipart stream lied about its size — the actual bytes
		// overflowed the cap.
		response.BadRequest(c, "File vượt quá giới hạn dung lượng")
		return nil, "", false
	}
	return fileBytes, fileHeader.Filename, true
}

func isBodyTooLarge(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "request body too large") ||
		strings.Contains(msg, "http: request body too large")
}

// excelMagic reports whether the head bytes look like an Excel workbook:
// a ZIP local file header (xlsx) and, when allowXLS, an OLE2 compound
// document (legacy .xls).
func excelMagic(head []byte, allowXLS bool) bool {
	if isZipMagic(head) {
		return true
	}
	return allowXLS && isOLE2Magic(head)
}

// isZipMagic reports whether the head bytes match a ZIP local file header
// (PK\x03\x04 / PK\x05\x06 / PK\x07\x08).
func isZipMagic(head []byte) bool {
	if len(head) < 4 {
		return false
	}
	return head[0] == 0x50 && head[1] == 0x4B &&
		(head[2] == 0x03 || head[2] == 0x05 || head[2] == 0x07) &&
		(head[3] == 0x04 || head[3] == 0x06 || head[3] == 0x08)
}

// isOLE2Magic reports whether the head bytes match the OLE2 compound
// document signature (legacy .xls).
func isOLE2Magic(head []byte) bool {
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
