package caskadht

import (
	"net/http"
	"strings"

	"github.com/ipni/go-libipni/apierror"
	"github.com/ipni/go-libipni/rwriter"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multicodec"
)

var drProtocolBitswap = multicodec.TransportBitswap.String()

const drSchemaBitswap = "bitswap"

type (
	delegatedRoutingLookupResponseWriter struct {
		rwriter.ResponseWriter
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

func newDelegatedRoutingLookupResponseWriter(w http.ResponseWriter, r *http.Request, preferJson bool) (*delegatedRoutingLookupResponseWriter, error) {
	if !strings.HasPrefix(r.URL.Path, "/routing/v1/providers/") {
		return nil, apierror.New(nil, http.StatusNotFound)
	}
	rspWriter, err := rwriter.New(w, r,
		rwriter.WithPreferJson(preferJson),
		rwriter.WithMultihashPathType(""),
		rwriter.WithCidPathType("providers"),
	)
	if err != nil {
		return nil, err
	}
	return &delegatedRoutingLookupResponseWriter{
		ResponseWriter: *rspWriter,
	}, nil
}

func (d *delegatedRoutingLookupResponseWriter) writeDrProviderRecord(provider peer.AddrInfo) error {
	rec := drProviderRecord{
		Protocol: drProtocolBitswap,
		Schema:   drSchemaBitswap,
		ID:       provider.ID,
		Addrs:    provider.Addrs,
	}
	if d.IsND() {
		if err := d.Encoder().Encode(rec); err != nil {
			logger.Errorw("Failed to encode ndjson response", "err", err)
			return err
		}
		d.Flush()
	} else {
		d.result.Providers = append(d.result.Providers, rec)
	}
	return nil
}

func (d *delegatedRoutingLookupResponseWriter) close() error {
	if d.IsND() {
		return nil
	}
	return d.Encoder().Encode(d.result)
}
