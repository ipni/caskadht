package cascadht

import (
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func Test_IsPubliclyDialableAddr(t *testing.T) {
	tests := []struct {
		name  string
		given []string
		want  []string
	}{
		{
			name: "nil",
		},
		{
			name:  "empty",
			given: []string{},
		},
		{
			name:  "bind addr",
			given: []string{"/ip4/0.0.0.0/"},
		},
		{
			name: "dns4",
			given: []string{
				"/dns4/example.invalid",
				"/ip4/0.0.0.0/",
			},
			want: []string{"/dns4/example.invalid"},
		},
		{
			name: "mix",
			given: []string{
				"/ip6/2604:1380:1000:6000::1/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dnsaddr/sjc-1.bootstrap.libp2p.io/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dnsaddr/localhost/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/ip4/147.75.83.83/tcp/4001/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
				"/ip4/127.0.0.1/tcp/4001",
				"/ip4/127.0.0.1/udp/4001",
			},
			want: []string{
				"/ip6/2604:1380:1000:6000::1/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dnsaddr/sjc-1.bootstrap.libp2p.io/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/ip4/147.75.83.83/tcp/4001/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			},
		},
		{
			name: "dns localhost",
			given: []string{
				"/dns/localhost/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dns4/localhost/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dns6/localhost/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dnsaddr/localhost/tcp/4001/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			givenAddrs := make([]multiaddr.Multiaddr, 0, len(tt.given))
			for _, a := range tt.given {
				addr, err := multiaddr.NewMultiaddr(a)
				require.NoError(t, err)
				givenAddrs = append(givenAddrs, addr)
			}
			gotAddrs := multiaddr.FilterAddrs(givenAddrs, IsPubliclyDialableAddr)
			if tt.want == nil {
				require.Empty(t, gotAddrs)
			} else {
				require.Equal(t, len(tt.want), len(gotAddrs))
				for i, addr := range gotAddrs {
					require.Equal(t, tt.want[i], addr.String())
				}
			}
		})
	}
}
