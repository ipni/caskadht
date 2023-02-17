package cascadht

import (
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// IsPubliclyDialableAddr checks weather target can be dialled publicly. More specifically:
//   - if it is of type IP, it is a public IP, and
//   - if it is of type DNS, it is not localhost
//
// All other address types are treated as dialable.
func IsPubliclyDialableAddr(target multiaddr.Multiaddr) bool {
	c, _ := multiaddr.SplitFirst(target)
	if c == nil {
		return false
	}
	switch c.Protocol().Code {
	case multiaddr.P_IP4, multiaddr.P_IP6, multiaddr.P_IP6ZONE, multiaddr.P_IPCIDR:
		return manet.IsPublicAddr(target)
	case multiaddr.P_DNS, multiaddr.P_DNS4, multiaddr.P_DNS6, multiaddr.P_DNSADDR:
		return c.Value() != "localhost"
	default:
		return true
	}
}
