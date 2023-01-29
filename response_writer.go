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
	selectiveResponseWriter interface {
		http.ResponseWriter
		Accept(r *http.Request) error
	}
	lookupResponseWriter interface {
		io.Closer
		selectiveResponseWriter
		Key() cid.Cid
		WriteProviderRecord(providerRecord) error
	}
)
