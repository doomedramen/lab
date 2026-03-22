package middleware

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// contextKey is a custom type for context keys
type ipContextKey string

const clientIPKey ipContextKey = "client_ip"

// IPWhitelistInterceptor provides IP-based access control for Connect RPC.
// It checks the client IP against a list of allowed CIDRs before allowing
// the request to proceed.
type IPWhitelistInterceptor struct {
	allowedNets []*net.IPNet
	enabled     bool
	publicPaths map[string]bool
}

// NewIPWhitelistInterceptor creates a new IP whitelist interceptor.
// If allowedCIDRs is empty, all IPs are allowed (whitelist disabled).
func NewIPWhitelistInterceptor(allowedCIDRs []string) (*IPWhitelistInterceptor, error) {
	// Define paths that bypass IP whitelist (e.g., health checks)
	publicPaths := map[string]bool{
		"/health":  true,
		"/healthz": true,
		"/readyz":  true,
		"/livez":   true,
		"/metrics": true, // Often scraped from monitoring systems
	}

	// If no CIDRs configured, whitelist is disabled
	if len(allowedCIDRs) == 0 {
		return &IPWhitelistInterceptor{
			enabled:     false,
			publicPaths: publicPaths,
		}, nil
	}

	// Parse CIDRs
	networks := make([]*net.IPNet, 0, len(allowedCIDRs))
	for _, cidr := range allowedCIDRs {
		// Handle single IP addresses (add /32 or /128)
		if !strings.Contains(cidr, "/") {
			ip := net.ParseIP(cidr)
			if ip == nil {
				return nil, errors.New("invalid IP address: " + cidr)
			}
			if ip.To4() != nil {
				cidr = cidr + "/32"
			} else {
				cidr = cidr + "/128"
			}
		}

		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, errors.New("invalid CIDR " + cidr + ": " + err.Error())
		}
		networks = append(networks, network)
	}

	return &IPWhitelistInterceptor{
		allowedNets: networks,
		enabled:     true,
		publicPaths: publicPaths,
	}, nil
}

// WrapUnary wraps unary Connect RPC calls with IP whitelist checking.
func (i *IPWhitelistInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Skip if whitelist is disabled
		if !i.enabled {
			return next(ctx, req)
		}

		// Check if path is public
		if i.isPublicPath(req.Spec().Procedure) {
			return next(ctx, req)
		}

		// Get client IP from context (set by HTTP middleware) or from headers
		clientIP := GetClientIPFromContext(ctx)
		if clientIP == "" {
			clientIP = i.getClientIP(req.Header())
		}

		if clientIP == "" {
			log.Printf("[IPWhitelist] Could not determine client IP for request to %s", req.Spec().Procedure)
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
		}

		// Check if IP is allowed
		if !i.isIPAllowed(clientIP) {
			log.Printf("[IPWhitelist] Access denied for IP %s to %s", clientIP, req.Spec().Procedure)
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
		}

		// Add client IP to context for downstream handlers
		ctx = SetClientIPInContext(ctx, clientIP)
		return next(ctx, req)
	}
}

// WrapStreamingClient wraps streaming client-side Connect RPC calls.
func (i *IPWhitelistInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler wraps streaming handler-side Connect RPC calls.
func (i *IPWhitelistInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// Skip if whitelist is disabled
		if !i.enabled {
			return next(ctx, conn)
		}

		// Check if path is public
		if i.isPublicPath(conn.Spec().Procedure) {
			return next(ctx, conn)
		}

		// Get client IP from context (set by HTTP middleware) or from headers
		clientIP := GetClientIPFromContext(ctx)
		if clientIP == "" {
			clientIP = i.getClientIP(conn.RequestHeader())
		}

		if clientIP == "" {
			log.Printf("[IPWhitelist] Could not determine client IP for streaming request to %s", conn.Spec().Procedure)
			return connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
		}

		// Check if IP is allowed
		if !i.isIPAllowed(clientIP) {
			log.Printf("[IPWhitelist] Access denied for IP %s to %s", clientIP, conn.Spec().Procedure)
			return connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
		}

		// Add client IP to context for downstream handlers
		ctx = SetClientIPInContext(ctx, clientIP)
		return next(ctx, conn)
	}
}

// isPublicPath checks if a path is publicly accessible.
func (i *IPWhitelistInterceptor) isPublicPath(path string) bool {
	return i.publicPaths[path]
}

// getClientIP extracts the client IP from request headers.
// It checks X-Forwarded-For, X-Real-IP, and falls back to empty string.
func (i *IPWhitelistInterceptor) getClientIP(header http.Header) string {
	// Check X-Forwarded-For header (may contain multiple IPs, use first)
	if xff := header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return ""
}

// isIPAllowed checks if the given IP is in the allowed CIDRs.
func (i *IPWhitelistInterceptor) isIPAllowed(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, network := range i.allowedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// IsEnabled returns true if IP whitelisting is enabled.
func (i *IPWhitelistInterceptor) IsEnabled() bool {
	return i.enabled
}

// AllowedNetworks returns the list of allowed networks (for debugging/logging).
func (i *IPWhitelistInterceptor) AllowedNetworks() []string {
	networks := make([]string, len(i.allowedNets))
	for idx, net := range i.allowedNets {
		networks[idx] = net.String()
	}
	return networks
}

// SetClientIPInContext adds the client IP to the context.
func SetClientIPInContext(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// GetClientIPFromContext retrieves the client IP from context.
func GetClientIPFromContext(ctx context.Context) string {
	ip, ok := ctx.Value(clientIPKey).(string)
	if !ok {
		return ""
	}
	return ip
}

// IPWhitelistHTTP is an HTTP middleware that checks IP whitelist
// and adds the client IP to the context for Connect handlers.
// This should be used as a standard net/http middleware.
func IPWhitelistHTTP(allowedCIDRs []string) (func(http.Handler) http.Handler, error) {
	interceptor, err := NewIPWhitelistInterceptor(allowedCIDRs)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if whitelist is disabled
			if !interceptor.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is public
			if interceptor.isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Get client IP
			clientIP := getClientIPFromRequest(r)
			if clientIP == "" {
				log.Printf("[IPWhitelist] Could not determine client IP for request to %s", r.URL.Path)
				http.Error(w, "Access Denied", http.StatusForbidden)
				return
			}

			// Check if IP is allowed
			if !interceptor.isIPAllowed(clientIP) {
				log.Printf("[IPWhitelist] Access denied for IP %s to %s", clientIP, r.URL.Path)
				http.Error(w, "Access Denied", http.StatusForbidden)
				return
			}

			// Add client IP to context
			ctx := SetClientIPInContext(r.Context(), clientIP)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// getClientIPFromRequest extracts the client IP from an HTTP request.
// It checks X-Forwarded-For, X-Real-IP, and falls back to RemoteAddr.
func getClientIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs, use first)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, RemoteAddr might be just an IP
		return r.RemoteAddr
	}
	return ip
}
