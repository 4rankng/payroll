package llmchat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type stubAssetReader struct {
	meta AssetMeta
	err  error
}

func (s stubAssetReader) GetAsset(context.Context, uint) (AssetMeta, error) {
	return s.meta, s.err
}

func writeTestPNG(t *testing.T, dir string) string {
	t.Helper()
	// 1x1 transparent PNG — a real PNG header for the sniffer.
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d,
		'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15,
		0xc4, 0x89, 0, 0, 0, 0x0a, 'I', 'D', 'A', 'T', 0x78, 0x9c, 0x63, 0, 1,
		0, 0, 5, 0, 1, 0x0d, 0x0a, 0x2d, 0xb4, 0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82}
	path := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(path, png, 0o600); err != nil {
		t.Fatalf("write png: %v", err)
	}
	return path
}

func TestLoadImageRejectsForeignUpload(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPNG(t, dir)
	loader := NewAssetImages(
		stubAssetReader{meta: AssetMeta{FilePath: path, UploadedBy: 42}},
		func(p string) (string, error) { return p, nil },
	)
	if _, err := loader.LoadImageDataURL(context.Background(), Caller{UserID: 7, Role: RolePartner}, 1); !errors.Is(err, ErrAttachmentRejected) {
		t.Fatalf("foreign upload err = %v, want ErrAttachmentRejected", err)
	}
	// the uploader and an admin both pass the ownership gate
	if _, err := loader.LoadImageDataURL(context.Background(), Caller{UserID: 42, Role: RolePartner}, 1); err != nil {
		t.Fatalf("uploader rejected: %v", err)
	}
	if _, err := loader.LoadImageDataURL(context.Background(), Caller{UserID: 1, Role: RoleAdmin}, 1); err != nil {
		t.Fatalf("admin rejected: %v", err)
	}
}

func TestLoadImageRejectsNonImageContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.png")
	// Extension says PNG, bytes say text — the sniffer must win.
	if err := os.WriteFile(path, []byte("this is definitely not an image, just text"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	loader := NewAssetImages(
		stubAssetReader{meta: AssetMeta{FilePath: path, UploadedBy: 1}},
		func(p string) (string, error) { return p, nil },
	)
	if _, err := loader.LoadImageDataURL(context.Background(), Caller{UserID: 1, Role: RoleAdmin}, 1); err == nil {
		t.Fatal("text bytes with .png extension must be rejected by content sniffing")
	}
}

func TestLoadImageBuildsDataURL(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPNG(t, dir)
	loader := NewAssetImages(
		stubAssetReader{meta: AssetMeta{FilePath: path, UploadedBy: 1}},
		func(p string) (string, error) { return p, nil },
	)
	dataURL, err := loader.LoadImageDataURL(context.Background(), Caller{UserID: 1, Role: RoleAdmin}, 1)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if dataURL[:len("data:image/png;base64,")] != "data:image/png;base64," {
		t.Fatalf("data URL prefix wrong: %s", dataURL[:30])
	}
}
