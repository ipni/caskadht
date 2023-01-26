package cascadht

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipns"
	"github.com/ipfs/go-log/v2"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"github.com/multiformats/go-varint"
)

var (
	logger = log.Logger("caskadht")

	newline          = []byte("\n")
	cascadeContextID = []byte("ipfs-dht-cascade")
	cascadeMetadata  = varint.ToUvarint(uint64(multicodec.TransportBitswap))
)

type (
	Caskadht struct {
		*options
		std *dht.IpfsDHT
		acc *fullrt.FullRT
		s   *http.Server

		// Context and cancellation used to terminate streaming responses on shutdown.
		ctx    context.Context
		cancel context.CancelFunc
	}

	response struct {
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

const (
	ipfsProtocolPrefix = "/ipfs"

	mediaTypeNDJson = "application/x-ndjson"
	mediaTypeJson   = "application/json"
	mediaTypeAny    = "*/*"
)

func New(o ...Option) (*Caskadht, error) {
	opts, err := newOptions(o...)
	if err != nil {
		return nil, err
	}
	var c Caskadht
	c.options = opts
	c.s = &http.Server{
		Addr:    opts.httpListenAddr,
		Handler: c.serveMux(),
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.s.RegisterOnShutdown(c.cancel)
	return &c, nil
}

func (c *Caskadht) Start(ctx context.Context) error {
	var err error
	// TODO parameterize options
	c.std, err = dht.New(ctx, c.h, dht.Mode(dht.ModeClient), dht.BootstrapPeers(c.bootstrapPeers...))
	if err != nil {
		return err
	}

	if c.useAccDHT {
		// TODO: parameterize options
		c.acc, err = fullrt.NewFullRT(c.h, ipfsProtocolPrefix,
			fullrt.DHTOption(
				dht.BucketSize(20),
				dht.Validator(record.NamespacedValidator{
					"pk":   record.PublicKeyValidator{},
					"ipns": ipns.Validator{},
				}),
				dht.BootstrapPeers(c.bootstrapPeers...),
				dht.Mode(dht.ModeClient),
			))
		if err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", c.s.Addr)
	if err != nil {
		return err
	}
	go func() { _ = c.s.Serve(ln) }()
	logger.Infow("Server started", "addr", ln.Addr())
	return nil
}

func (c *Caskadht) serveMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/multihash/", c.handleMhSubtree)
	mux.HandleFunc("/ready", c.handleReady)
	mux.HandleFunc("/", c.handleCatchAll)
	return mux
}

func (c *Caskadht) handleMhSubtree(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleGetMh(w, r)
	default:
		discardBody(r)
		http.Error(w, "", http.StatusNotFound)
	}
}

func (c *Caskadht) handleGetMh(w http.ResponseWriter, r *http.Request) {
	discardBody(r)

	smh := strings.TrimPrefix(path.Base(r.URL.Path), "multihash/")
	logger := logger.With("mh", smh)
	mh, err := multihash.FromB58String(smh)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	okNDJson, okJson, err := acceptsJson(r)
	if err != nil {
		logger.Debugw("Failed to check accepted response media type", "err", err)
		http.Error(w, "invalid Accept header", http.StatusBadRequest)
		return
	}
	flusher, okFlusher := w.(http.Flusher)
	if !okFlusher && !okJson && okNDJson {
		// Respond with error if the request only accepts ndjson and the server does not support
		// streaming.
		http.Error(w, "server does not support streaming response", http.StatusBadRequest)
		return
	}
	if !okJson && !okNDJson {
		http.Error(w, "media type not supported", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	key := cid.NewCidV1(cid.Raw, mh)
	pch := c.cascadeFindProviders(ctx, key)
	// TODO: Decide response media type based on `q` weighting param in Accept.
	//       For now always prefer streaming responses.
	if !okFlusher && okJson {
		// We cannot stream results and the client accepts JSON; respond with non-streaming JSON.
		res := MultihashResult{
			Multihash: mh,
		}
	JSON_LOOP:
		for {
			select {
			case <-c.ctx.Done():
				break JSON_LOOP
			case provider, ok := <-pch:
				if !ok {
					break JSON_LOOP
				}
				res.ProviderResults = append(res.ProviderResults, ProviderResult{
					ContextID: cascadeContextID,
					Metadata:  cascadeMetadata,
					Provider:  provider,
				})
			}
		}
		cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{
			MultihashResults: []MultihashResult{res},
		}); err != nil {
			logger.Errorw("failed to write provider results", "count", len(res.ProviderResults), "err", err)
		}
		return
	}

	w.Header().Set("Content-Type", mediaTypeNDJson)
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	encoder := json.NewEncoder(w)
	var count int

NDJSON_LOOP:
	for {
		select {
		case <-c.ctx.Done():
			break NDJSON_LOOP
		case provider, ok := <-pch:
			if !ok {
				break NDJSON_LOOP
			}
			// TODO: Restructure response once there is a more optimal way to stream results.
			//       See: https://github.com/ipni/specs/issues/8
			if err := encoder.Encode(response{
				MultihashResults: []MultihashResult{
					{
						Multihash: mh,
						ProviderResults: []ProviderResult{
							{
								ContextID: cascadeContextID,
								Metadata:  cascadeMetadata,
								Provider:  provider,
							},
						},
					},
				},
			}); err != nil {
				logger.Errorw("Failed to encode ndjson response", "err", err)
				break NDJSON_LOOP
			}
			if _, err := w.Write(newline); err != nil {
				logger.Errorw("Failed to encode ndjson response", "err", err)
				break NDJSON_LOOP
			}
			flusher.Flush()
			count++
		}
	}
	cancel()
	logger.Debugw("Finished streaming results", "count", count)
	if count == 0 {
		http.Error(w, "", http.StatusNotFound)
	}
}

func acceptsJson(r *http.Request) (ndjson bool, json bool, err error) {
	accepts := r.Header.Values("Accept")
	var mt string
	for _, accept := range accepts {
		mt, _, err = mime.ParseMediaType(accept)
		if err != nil {
			return
		}
		switch mt {
		case mediaTypeNDJson:
			ndjson = true
		case mediaTypeJson:
			json = true
		case mediaTypeAny:
			ndjson = true
			json = true
		}
		if json && ndjson {
			// Return early if both is supported.
			return
		}
	}
	return
}

func (c *Caskadht) cascadeFindProviders(ctx context.Context, key cid.Cid) <-chan peer.AddrInfo {
	return c.contentRouting().FindProvidersAsync(ctx, key, 0)
}

func (c *Caskadht) contentRouting() routing.ContentRouting {
	if c.useAccDHT && c.acc.Ready() {
		return c.acc
	}
	return c.std
}

func (c *Caskadht) handleReady(w http.ResponseWriter, r *http.Request) {
	discardBody(r)
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "", http.StatusNotFound)
	}
}

func (c *Caskadht) handleCatchAll(w http.ResponseWriter, r *http.Request) {
	discardBody(r)
	http.Error(w, "", http.StatusNotFound)
}

func (c *Caskadht) Shutdown(ctx context.Context) error {
	sErr := c.s.Shutdown(ctx)
	_ = c.std.Close()
	if c.acc != nil {
		_ = c.acc.Close()
	}
	hErr := c.h.Close()

	switch {
	case sErr != nil:
		return sErr
	default:
		return hErr
	}
}

func discardBody(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
}
