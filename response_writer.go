package cascadht

import (
	"io"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type (
	providerRecord struct {
		peer.AddrInfo
	}
	acceptor interface {
		Accept(r *http.Request) error
	}
	lookupResponseWriter interface {
		io.Closer
		http.ResponseWriter
		acceptor
		Key() cid.Cid
		WriteProviderRecord(providerRecord) error
	}
)
