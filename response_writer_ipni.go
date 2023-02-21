package cascadht

import (
	"net/http"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

var _ lookupResponseWriter = (*ipniLookupResponseWriter)(nil)

const ipniCascadeQueryKey = "cascade"

type (
	ipniLookupResponseWriter struct {
		jsonResponseWriter
		result            MultihashResult
		count             int
		cascadeLabel      string
		requireQueryParam bool
	}
	ipniResults struct {
		MultihashResults []MultihashResult
	}
	MultihashResult struct {
		Multihash       multihash.Multihash
		ProviderResults []ProviderResult
	}
	ProviderResult struct {
		ContextID []byte
		Metadata  []byte
		Provider  peer.AddrInfo
	}
)

func newIPNILookupResponseWriter(w http.ResponseWriter, cascadeLabel string, requireQueryParam bool, preferJson bool) lookupResponseWriter {
	return &ipniLookupResponseWriter{
		jsonResponseWriter: newJsonResponseWriter(w, preferJson),
		cascadeLabel:       cascadeLabel,
		requireQueryParam:  requireQueryParam,
	}
}

func (i *ipniLookupResponseWriter) Accept(r *http.Request) error {
	if err := i.jsonResponseWriter.Accept(r); err != nil {
		return err
	}
	if i.requireQueryParam {
		if !r.URL.Query().Has(ipniCascadeQueryKey) {
			logger.Debugw("Rejected request with unspecified cascade query parameter.")
			return errHttpResponse{status: http.StatusNotFound}
		}
		if got := r.URL.Query().Get(ipniCascadeQueryKey); i.cascadeLabel != got {
			logger.Debugw("Rejected request with mismatching cascade label.", "want", i.cascadeLabel, "got", got)
			return errHttpResponse{status: http.StatusNotFound}
		}
	}

	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/cid/"):
		scid := strings.TrimPrefix(path, "/cid/")
		var err error
		c, err := cid.Decode(scid)
		if err != nil {
			return errHttpResponse{message: err.Error(), status: http.StatusBadRequest}
		}
		i.result.Multihash = c.Hash()
	case strings.HasPrefix(path, "/multihash/"):
		smh := strings.TrimPrefix(path, "/multihash/")
		var err error
		i.result.Multihash, err = multihash.FromB58String(smh)
		if err != nil {
			return errHttpResponse{message: err.Error(), status: http.StatusBadRequest}
		}
	default:
		return errHttpResponse{status: http.StatusNotFound}
	}
	return nil
}

func (i *ipniLookupResponseWriter) Key() cid.Cid {
	return cid.NewCidV1(cid.Raw, i.result.Multihash)
}

func (i *ipniLookupResponseWriter) WriteProviderRecord(provider providerRecord) error {
	rec := ProviderResult{
		ContextID: cascadeContextID,
		Metadata:  cascadeMetadata,
		Provider:  provider.AddrInfo,
	}
	if i.nd {
		if err := i.encoder.Encode(rec); err != nil {
			logger.Errorw("Failed to encode ndjson response", "err", err)
			return err
		}
		if _, err := i.w.Write(newline); err != nil {
			logger.Errorw("Failed to encode ndjson response", "err", err)
			return err
		}
		if i.f != nil {
			i.f.Flush()
		}
	} else {
		i.result.ProviderResults = append(i.result.ProviderResults, rec)
	}
	i.count++
	return nil
}

func (i *ipniLookupResponseWriter) Close() error {
	if i.count == 0 {
		return errHttpResponse{status: http.StatusNotFound}
	}
	logger.Debugw("Finished writing ipni results", "count", i.count)
	if i.nd {
		return nil
	}
	return i.encoder.Encode(ipniResults{
		MultihashResults: []MultihashResult{i.result},
	})
}
