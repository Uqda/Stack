// Package safebind enforces secure defaults for local proxy listeners.
package safebind

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ValidateSOCKS rejects wildcard and non-loopback TCP listeners unless the
// operator has explicitly allowed a public proxy. Paths without a colon are
// treated as Unix-domain sockets and are local by construction.
func ValidateSOCKS(endpoint string, allowPublic bool) error {
	if endpoint == "" || !strings.Contains(endpoint, ":") {
		return nil
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("invalid SOCKS listen address %q: %w", endpoint, err)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid SOCKS port %q", port)
	}
	if allowPublic {
		return nil
	}
	ip := net.ParseIP(host)
	if host == "" || (ip == nil && !strings.EqualFold(host, "localhost")) || (ip != nil && !ip.IsLoopback()) {
		return fmt.Errorf("refusing non-loopback SOCKS listener %q; bind to 127.0.0.1/[::1] or explicitly use -allow-public-socks", endpoint)
	}
	return nil
}
