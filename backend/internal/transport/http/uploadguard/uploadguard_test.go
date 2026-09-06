package uploadguard

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func init() { gin.SetMode(gin.TestMode) }

// A real (tiny) xlsx passes Validate + Receive.
func TestGuardAcceptsValidXlsx(t *testing.T) {
	src := excelize.NewFile()
	if err := src.SetCellValue("Sheet1", "A1", "ok"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := src.Write(&buf); err != nil {
		t.Fatal(err)
	}

	c := uploadContext(t, "test.xlsx", buf.Bytes())
	if _, ok := Validate(c, false); !ok {
		t.Fatal("valid xlsx rejected")
	}

	c = uploadContext(t, "test.xlsx", buf.Bytes())
	got, name, ok := Receive(c, false)
	if !ok || len(got) != buf.Len() || name != "test.xlsx" {
		t.Fatalf("Receive = %d bytes, name=%q, ok=%v; want %d, test.xlsx, true", len(got), name, ok, buf.Len())
	}
}

func TestGuardRejectsGarbage(t *testing.T) {
	c := uploadContext(t, "fake.xlsx", []byte("this is not a zip"))
	if _, ok := Validate(c, false); ok {
		t.Fatal("garbage accepted as xlsx")
	}
	if c.Writer.Status() != 400 {
		t.Fatalf("status = %d, want 400", c.Writer.Status())
	}
}

// Legacy .xls (OLE2) is accepted only when the endpoint allows it.
func TestGuardOLE2Gating(t *testing.T) {
	ole2 := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0, 0, 0}

	c := uploadContext(t, "legacy.xls", ole2)
	if _, ok := Validate(c, false); ok {
		t.Fatal("OLE2 accepted on xlsx-only endpoint")
	}

	c = uploadContext(t, "legacy.xls", ole2)
	if _, ok := Validate(c, true); !ok {
		t.Fatal("OLE2 rejected on xls-allowing endpoint")
	}
}

func TestGuardRejectsMissingFile(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	if _, ok := Validate(c, false); ok {
		t.Fatal("missing file accepted")
	}
}

func uploadContext(t *testing.T, filename string, content []byte) *gin.Context {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestExcelMagic(t *testing.T) {
	if !isZipMagic([]byte{0x50, 0x4B, 0x03, 0x04}) {
		t.Error("ZIP magic not detected")
	}
	if isZipMagic([]byte{0x50, 0x4B, 0x09, 0x04}) {
		t.Error("non-ZIP PK prefix accepted")
	}
	if !isOLE2Magic([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		t.Error("OLE2 magic not detected")
	}
}
