package email

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
)

// BrandBannerCID is the Content-ID under which the shared brand banner is
// referenced from email HTML via <img src="cid:brand-banner">.
const BrandBannerCID = "brand-banner"

// BrandingSender is an EmailDeliveryPort decorator that attaches the shared
// brand banner to any message whose HTML references it via the banner CID.
// Templates opt in by referencing the CID; this decorator supplies the image
// bytes (loaded lazily and cached once) so every outbound path — OTP and
// notifications alike — gets the banner without duplicating attach logic.
//
// If the banner asset is unavailable the send still proceeds (logged once); a
// missing brand asset must never block email delivery.
type BrandingSender struct {
	inner  domain.EmailDeliveryPort
	logger *slog.Logger
}

// NewBrandingSender wraps inner so messages that reference the brand banner CID
// automatically receive the inline banner attachment before delivery.
func NewBrandingSender(inner domain.EmailDeliveryPort, logger *slog.Logger) *BrandingSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &BrandingSender{inner: inner, logger: logger}
}

// Send delegates to the inner provider, first attaching the brand banner when
// the message HTML references its CID and no inline attachment is present yet.
func (b *BrandingSender) Send(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	if msg != nil && strings.Contains(msg.HTMLBody, "cid:"+BrandBannerCID) && !hasInlineAttachment(msg, BrandBannerCID) {
		if banner := loadBannerBytes(b.logger); banner != nil {
			msg.Attachments = append(msg.Attachments, domain.EmailAttachment{
				Filename:    "banner_inline.jpg",
				ContentType: "image/jpeg",
				Content:     banner,
				ContentID:   BrandBannerCID,
			})
		}
	}
	return b.inner.Send(ctx, msg)
}

func hasInlineAttachment(msg *domain.EmailMessage, cid string) bool {
	for _, a := range msg.Attachments {
		if a.ContentID == cid {
			return true
		}
	}
	return false
}

var (
	bannerOnce sync.Once
	bannerData []byte
)

// loadBannerBytes loads the brand banner from disk once and caches it for the
// process lifetime. Returns nil (logging the first failure only) when the
// asset is unavailable so callers can gracefully skip attaching it.
func loadBannerBytes(logger *slog.Logger) []byte {
	bannerOnce.Do(func() {
		data, err := os.ReadFile(pkgConstants.BrandBannerPath)
		if err != nil {
			if logger != nil {
				logger.Warn("brand banner asset not loaded; emails will omit the logo",
					"path", pkgConstants.BrandBannerPath, "error", err)
			}
			return
		}
		bannerData = data
	})
	return bannerData
}
