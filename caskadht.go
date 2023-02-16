package cascadht

import (
	"context"
	"io"
	"net"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipns"
	"github.com/ipfs/go-log/v2"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-varint"
)

var (
	logger = log.Logger("caskadht")

	newline          = []byte("\n")
	cascadeContextID = []byte("ipfs-dht-cascade")
	cascadeMetadata  = varint.ToUvarint(uint64(multicodec.TransportBitswap))
)

type Caskadht struct {
	*options
	std *dht.IpfsDHT
	acc *fullrt.FullRT
	s   *http.Server

	// Context and cancellation used to terminate streaming responses on shutdown.
	ctx    context.Context
	cancel context.CancelFunc
}

const ipfsProtocolPrefix = "/ipfs"

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
	logger.Infow("Server started", "id", c.h.ID(), "libp2pAddrs", c.h.Addrs(), "httpAddr", ln.Addr())
	return nil
}

func (c *Caskadht) serveMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/multihash", c.handleMh)
	mux.HandleFunc("/multihash/", c.handleMhSubtree)
	mux.HandleFunc("/routing/v1/providers/", c.handleRoutingV1ProvidersSubtree)
	mux.HandleFunc("/ready", c.handleReady)
	mux.HandleFunc("/", c.handleCatchAll)
	return mux
}

func (c *Caskadht) handleMh(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		discardBody(r)
		c.handleLookupOptions(w)
	default:
		discardBody(r)
		http.Error(w, "", http.StatusNotFound)
	}
}

func (c *Caskadht) handleMhSubtree(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleLookup(newIPNILookupResponseWriter(w, c.ipniCascadeLabel, c.ipniRequireCascadeQueryParam, c.httpResponsePreferJson), r)
	case http.MethodOptions:
		discardBody(r)
		c.handleLookupOptions(w)
	default:
		discardBody(r)
		http.Error(w, "", http.StatusNotFound)
	}
}

func (c *Caskadht) handleRoutingV1ProvidersSubtree(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleLookup(newDelegatedRoutingLookupResponseWriter(w, c.httpResponsePreferJson), r)
	case http.MethodOptions:
		discardBody(r)
		c.handleLookupOptions(w)
	case http.MethodPut:
		discardBody(r)
		http.Error(w, "", http.StatusNotImplemented)
	default:
		discardBody(r)
		http.Error(w, "", http.StatusNotFound)
	}
}

func (c *Caskadht) handleLookup(w lookupResponseWriter, r *http.Request) {
	if err := w.Accept(r); err != nil {
		switch e := err.(type) {
		case errHttpResponse:
			e.WriteTo(w)
		default:
			logger.Errorw("Failed to accept lookup request", "err", err)
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	pch := c.cascadeFindProviders(ctx, w.Key())
	defer cancel()
LOOP:
	for {
		select {
		case <-c.ctx.Done():
			logger.Debugw("Interrupted while responding to lookup", "key", w.Key(), "err", ctx.Err())
			break LOOP
		case provider, ok := <-pch:
			if !ok {
				logger.Debugw("No more provider records", "key", w.Key())
				break LOOP
			}

			if err := provider.ID.Validate(); err != nil {
				logger.Debugw("Skipping provider record with invalid ID", "err", err)
				continue
			}
			if len(provider.Addrs) == 0 {
				found, err := c.routing().FindPeer(ctx, provider.ID)
				if err != nil {
					logger.Errorw("Failed to discover addrs for peer ID; skipping provider.", "id", provider.ID, "err", err)
					continue
				}
				if len(found.Addrs) == 0 {
					logger.Debugw("Found no addrs for peer ID; skipping provider", "id", provider.ID)
					continue
				}
				provider.Addrs = found.Addrs
			}

			if !c.addrFilterDisabled {
				provider.Addrs = multiaddr.FilterAddrs(provider.Addrs, manet.IsPublicAddr)
			}

			if len(provider.Addrs) == 0 {
				logger.Debugw("Found no public addrs for peer ID; skipping provider", "id", provider.ID)
				continue
			}

			if err := w.WriteProviderRecord(providerRecord{AddrInfo: provider}); err != nil {
				logger.Errorw("Failed to encode provider record", "err", err)
				break LOOP
			}
		}
	}
	if err := w.Close(); err != nil {
		switch e := err.(type) {
		case errHttpResponse:
			e.WriteTo(w)
		default:
			logger.Errorw("Failed to finalize lookup results", "err", err)
			http.Error(w, "", http.StatusInternalServerError)
		}
	}
}

func (c *Caskadht) cascadeFindProviders(ctx context.Context, key cid.Cid) <-chan peer.AddrInfo {
	return c.routing().FindProvidersAsync(ctx, key, 0)
}

func (c *Caskadht) routing() routing.Routing {
	if c.useAccDHT && c.acc.Ready() {
		return c.acc
	}
	return c.std
}

func (c *Caskadht) handleLookupOptions(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", c.httpAllowOrigin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("X-IPNI-Allow-Cascade", c.ipniCascadeLabel)
	w.WriteHeader(http.StatusAccepted)
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
