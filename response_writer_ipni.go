package cascadht

import (
	"net/http"
	"path"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

var _ lookupResponseWriter = (*ipniLookupResponseWriter)(nil)

type (
	ipniLookupResponseWriter struct {
		jsonAcceptor
		result MultihashResult
		count  int
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

func newIPNILookupResponseWriter(w http.ResponseWriter) lookupResponseWriter {
	return &ipniLookupResponseWriter{
		jsonAcceptor: newJsonAcceptor(w),
	}
}

func (i *ipniLookupResponseWriter) Accept(r *http.Request) error {
	if err := i.jsonAcceptor.Accept(r); err != nil {
		return err
	}
	smh := strings.TrimPrefix(path.Base(r.URL.Path), "multihash/")
	var err error
	i.result.Multihash, err = multihash.FromB58String(smh)
	if err != nil {
		return errHttpResponse{message: err.Error(), status: http.StatusBadRequest}
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
		if err := i.encoder.Encode(ipniResults{
			MultihashResults: []MultihashResult{
				{
					Multihash:       i.result.Multihash,
					ProviderResults: []ProviderResult{rec},
				},
			},
		}); err != nil {
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
		i.count++
	}
	i.result.ProviderResults = append(i.result.ProviderResults, rec)
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
