package caskadht

import (
	"net/http"
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
		jsonResponseWriter
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

func newDelegatedRoutingLookupResponseWriter(w http.ResponseWriter, preferJson bool) lookupResponseWriter {
	return &delegatedRoutingLookupResponseWriter{
		jsonResponseWriter: newJsonResponseWriter(w, preferJson),
	}
}

func (d *delegatedRoutingLookupResponseWriter) Accept(r *http.Request) error {
	if err := d.jsonResponseWriter.Accept(r); err != nil {
		return err
	}
	if !strings.HasPrefix(r.URL.Path, "/routing/v1/providers/") {
		return errHttpResponse{status: http.StatusNotFound}
	}
	sc := strings.TrimPrefix(r.URL.Path, "/routing/v1/providers/")
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
	} else {
		d.result.Providers = append(d.result.Providers, rec)
	}
	return nil
}

func (d *delegatedRoutingLookupResponseWriter) Close() error {
	if d.nd {
		return nil
	}
	return d.encoder.Encode(d.result)
}
