package cascadht

import (
	"net/http"
	"path"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multicodec"
)

var (
	_ lookupResponseWriter = (*delegatedRoutingLookupResponseWriter)(nil)

	drProtocolBitswap = multicodec.TransportBitswap.String()
)

const drSchemaBitswap = "bitswap"

type (
	delegatedRoutingLookupResponseWriter struct {
		jsonAcceptor
		key    cid.Cid
		result drProviderRecords
	}
	drProviderRecords struct {
		Providers []drProviderRecord
	}
	drProviderRecord struct {
		Protocol string
		Schema   string
		ID       peer.ID
		Addrs    []multiaddr.Multiaddr
	}
)

func newDelegatedRoutingLookupResponseWriter(w http.ResponseWriter) lookupResponseWriter {
	return &delegatedRoutingLookupResponseWriter{
		jsonAcceptor: newJsonAcceptor(w),
	}
}

func (d *delegatedRoutingLookupResponseWriter) Accept(r *http.Request) error {
	if err := d.jsonAcceptor.Accept(r); err != nil {
		return err
	}
	sc := strings.TrimPrefix(path.Base(r.URL.Path), "routing/v1/providers/")
	var err error
	d.key, err = cid.Decode(sc)
	if err != nil {
		return errHttpResponse{message: err.Error(), status: http.StatusBadRequest}
	}
	return nil
}
func (d *delegatedRoutingLookupResponseWriter) Key() cid.Cid {
	return d.key
}

func (d *delegatedRoutingLookupResponseWriter) WriteProviderRecord(provider providerRecord) error {
	rec := drProviderRecord{
		Protocol: drProtocolBitswap,
		Schema:   drSchemaBitswap,
		ID:       provider.ID,
		Addrs:    provider.Addrs,
	}
	if d.nd {
		if err := d.encoder.Encode(rec); err != nil {
			logger.Errorw("Failed to encode ndjson response", "err", err)
			return err
		}
		if _, err := d.w.Write(newline); err != nil {
			logger.Errorw("Failed to encode ndjson response", "err", err)
			return err
		}
		if d.f != nil {
			d.f.Flush()
		}
	}
	d.result.Providers = append(d.result.Providers, rec)
	return nil
}

func (d *delegatedRoutingLookupResponseWriter) Close() error {
	if d.nd {
		return nil
	}
	return d.encoder.Encode(d.result)
}
