package offensive

import (
	"fmt"
	"net/netip"
)

// RulesOfEngagement enforces the boundaries of offensive operations.
type RulesOfEngagement struct {
	allowedPrefixes []netip.Prefix
}

// NewRulesOfEngagement initializes the default Purple Team RoE.
// Restricts targets exclusively to RFC 1918 (Private IPv4),
// Link-Local, and localhost to prevent unauthorized external attacks.
func NewRulesOfEngagement() *RulesOfEngagement {
	// Must compile these prefixes successfully, otherwise panic on boot for safety.
	return &RulesOfEngagement{
		allowedPrefixes: []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/8"),     // RFC 1918
			netip.MustParsePrefix("172.16.0.0/12"),  // RFC 1918
			netip.MustParsePrefix("192.168.0.0/16"), // RFC 1918
			netip.MustParsePrefix("127.0.0.0/8"),    // Localhost
			netip.MustParsePrefix("169.254.0.0/16"), // Link-local
			netip.MustParsePrefix("::1/128"),        // IPv6 Loopback
			netip.MustParsePrefix("fe80::/10"),      // IPv6 Link-local
			netip.MustParsePrefix("fc00::/7"),       // IPv6 Unique Local
		},
	}
}

// IsTargetAllowed parses the target IP address and validates it against the allowed prefixes.
// Returns an error if the target is public, unparseable, or disallowed by RoE.
func (roe *RulesOfEngagement) IsTargetAllowed(target string) error {
	addr, err := netip.ParseAddr(target)
	if err != nil {
		return fmt.Errorf("[RoE VIOLATION] Unparseable IP address or hostname resolution strictly disallowed: %s", target)
	}

	for _, prefix := range roe.allowedPrefixes {
		if prefix.Contains(addr) {
			return nil // Allowed
		}
	}

	return fmt.Errorf("[RoE VIOLATION] Target IP %s is outside allowed Purple Team boundaries (RFC 1918). Offensive operation ABORTED", target)
}
