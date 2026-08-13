package middleware

import (
	"net"

	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Option configures IPWhitelist behavior.
type Option func(*whitelistConfig)

type whitelistConfig struct {
	useRemoteAddr bool
}

// UseRemoteAddr forces the middleware to read the client IP from
// c.Request.RemoteAddr (the raw TCP socket peer) instead of c.ClientIP().
// Required when running behind a reverse proxy/load-balancer that rewrites
// X-Forwarded-For, because gin's ClientIP() resolves to a trusted-proxy IP
// in that case and the whitelist would reject every legitimate webhook.
// Pair with a server-side trust boundary that scrubs inbound X-Forwarded-For
// before the request reaches this middleware.
func UseRemoteAddr() Option {
	return func(c *whitelistConfig) { c.useRemoteAddr = true }
}

// IPWhitelist returns a Gin middleware that rejects requests whose client
// IP is not in the allowed list. An empty list is a no-op (all requests pass).
//
// IMPORTANT — proxy coupling: by default, client IP comes from c.ClientIP(),
// which honours gin's SetTrustedProxies configuration. If the service runs
// behind an L7 proxy and TRUSTED_PROXIES is not configured to include that
// proxy's CIDR, every request resolves to the proxy IP and the whitelist
// will reject all traffic. If the proxy is trusted but client-supplied
// X-Forwarded-For is forwarded unfiltered, the header can be spoofed and
// the whitelist becomes meaningless. In either deployment, pass the
// UseRemoteAddr() option so the middleware uses the raw socket IP.
func IPWhitelist(allowedIPs []string, opts ...Option) gin.HandlerFunc {
	cfg := whitelistConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	exactIPs, networks := parseAllowedIPRanges(allowedIPs)
	return func(c *gin.Context) {
		if len(allowedIPs) == 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if cfg.useRemoteAddr {
			host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				host = c.Request.RemoteAddr
			}
			clientIP = host
		}

		if isAllowedIP(clientIP, exactIPs, networks) {
			c.Next()
			return
		}

		observability.GetLogger().Warn("IP whitelist blocked request",
			"client_ip", clientIP,
			"path", c.Request.URL.Path,
		)
		response.Forbidden(c, "ip not allowed")
		c.Abort()
	}
}

func parseAllowedIPRanges(entries []string) (map[string]struct{}, []*net.IPNet) {
	exactIPs := make(map[string]struct{}, len(entries))
	networks := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		if ip := net.ParseIP(entry); ip != nil {
			exactIPs[ip.String()] = struct{}{}
			continue
		}

		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			observability.GetLogger().Warn("ignoring invalid IP whitelist entry", "entry", entry)
			continue
		}
		networks = append(networks, network)
	}
	return exactIPs, networks
}

func isAllowedIP(clientIP string, exactIPs map[string]struct{}, networks []*net.IPNet) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	if _, ok := exactIPs[ip.String()]; ok {
		return true
	}
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
