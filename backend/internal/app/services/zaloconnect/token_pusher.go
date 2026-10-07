package zaloconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"api-server/internal/infra/zalo"
)

// tokenPushTimeout bounds one webhook delivery. The Provider invokes the hook
// inside the send/renew path, so the push must never hang a send.
const tokenPushTimeout = 5 * time.Second

// TokenWebhookPusher hands the current OA access token to the TingTing
// chatbot. Payroll is the sole rotator of the shared Zalo OA token pair: the
// chatbot pulls the token via its integration endpoint and additionally
// receives a push after every rotation and once at application startup. The
// refresh token NEVER leaves payroll.
//
// Delivery is fire-and-forget: Push returns nothing and logs failures at WARN,
// so a dead chatbot can never fail a ZNS send or the daily token renewal.
type TokenWebhookPusher struct {
	url    string
	apiKey string
	client *http.Client
	log    *slog.Logger
}

// NewTokenWebhookPusher constructs the pusher from the CHATBOT_OA_TOKEN_WEBHOOK_URL
// and CHATBOT_OA_TOKEN_WEBHOOK_KEY env-backed config. Both are optional: with
// either unset the pusher skips at debug, keeping unconfigured deployments inert.
func NewTokenWebhookPusher(url, apiKey string, log *slog.Logger) *TokenWebhookPusher {
	if log == nil {
		log = slog.Default()
	}
	return &TokenWebhookPusher{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{Timeout: tokenPushTimeout},
		log:    log,
	}
}

// tokenWebhookPayload is the JSON body the chatbot webhook receives.
type tokenWebhookPayload struct {
	AccessToken string `json:"access_token"`
}

// Push delivers accessToken to the configured chatbot webhook. Never returns
// an error and never logs the token.
func (p *TokenWebhookPusher) Push(ctx context.Context, accessToken string) {
	if accessToken == "" {
		return
	}
	if p.url == "" || p.apiKey == "" {
		p.log.Debug("zalo: chatbot token webhook not fully configured, skipping push")
		return
	}
	body, err := json.Marshal(tokenWebhookPayload{AccessToken: accessToken})
	if err != nil {
		p.log.Warn("zalo: encode chatbot token push payload", "error", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		p.log.Warn("zalo: build chatbot token push request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warn("zalo: chatbot token webhook push failed", "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		p.log.Warn("zalo: chatbot token webhook rejected the push", "http_status", resp.StatusCode)
	}
}

// PushCurrent backs the startup push: it reads the credential source and hands
// the freshly booted process's stored access token to the chatbot, so the bot
// holds the latest token before any rotation can run. Fails soft throughout.
func (p *TokenWebhookPusher) PushCurrent(ctx context.Context, creds zalo.CredentialSource) {
	current, err := creds.Get(ctx)
	if err != nil {
		p.log.Warn("zalo: startup token push aborted, cannot read credentials", "error", err)
		return
	}
	if current.AccessToken == "" {
		p.log.Info("zalo: startup token push skipped, no access token configured")
		return
	}
	p.Push(ctx, current.AccessToken)
}
