package llmchat

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// AssetImages reads chat screenshot attachments from the asset store.
// Ownership is enforced here: the uploader or an admin — nobody else's
// screenshot may enter another user's prompt.
type AssetImages struct {
	assets   AssetReader
	fullPath func(filePath string) (string, error)
}

// AssetReader is the asset metadata surface the loader needs.
type AssetReader interface {
	GetAsset(ctx context.Context, id uint) (AssetMeta, error)
}

// AssetMeta is the narrow asset projection the loader reads.
type AssetMeta struct {
	FilePath   string
	UploadedBy uint
}

// MaxImageBytes bounds one screenshot (5 MB).
const MaxImageBytes = 5 << 20

// NewAssetImages constructs the loader. fullPath resolves an asset file path
// to its absolute location (the asset service's CheckFileExists).
func NewAssetImages(assets AssetReader, fullPath func(string) (string, error)) *AssetImages {
	return &AssetImages{assets: assets, fullPath: fullPath}
}

// LoadImageDataURL returns a base64 data URL for the image. The content type
// is sniffed from the bytes — the stored extension is never trusted.
func (l *AssetImages) LoadImageDataURL(ctx context.Context, caller Caller, assetID uint) (string, error) {
	asset, err := l.assets.GetAsset(ctx, assetID)
	if err != nil {
		return "", ErrAttachmentRejected
	}
	if caller.Role != RoleAdmin && asset.UploadedBy != caller.UserID {
		return "", ErrAttachmentRejected
	}
	abs, err := l.fullPath(asset.FilePath)
	if err != nil {
		return "", ErrAttachmentRejected
	}
	data, err := os.ReadFile(filepath.Clean(abs))
	if err != nil || len(data) == 0 || len(data) > MaxImageBytes {
		return "", ErrAttachmentRejected
	}
	mime := http.DetectContentType(data[:512])
	if mime != "image/png" && mime != "image/jpeg" {
		return "", fmt.Errorf("%w: chỉ nhận ảnh PNG hoặc JPG", ErrAttachmentRejected)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
