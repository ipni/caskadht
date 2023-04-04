package caskadht

import (
	"context"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric"
)

const (
	meterName                  = "ipni/caskadht"
	meterLookupRespTTFP        = meterName + "/lookup_response_first_provider_time"
	meterLookupRespResultCount = meterName + "/lookup_response_result_count"
	meterLookupRespLatency     = meterName + "/lookup_response_latency"
	meterLookupReqCount        = meterName + "/lookup_request_count"
)

var meterScope = instrumentation.Scope{Name: meterName}

type metrics struct {
	c        *Caskadht
	server   *http.Server
	exporter *prometheus.Exporter

	lookupRequestCounter               instrument.Int64Counter
	lookupResponseTTFPHistogram        instrument.Int64Histogram
	lookupResponseResultCountHistogram instrument.Int64Histogram
	lookupResponseLatencyHistogram     instrument.Int64Histogram
}

func newMetrics(c *Caskadht) (*metrics, error) {
	m := metrics{
		c: c,
		server: &http.Server{
			Addr: c.metricsHttpListenAddr,
			// TODO add other metrics server options.
		},
	}
	return &m, nil
}

func (m *metrics) Start(_ context.Context) error {
	var err error
	if m.exporter, err = prometheus.New(
		prometheus.WithoutUnits(),
		prometheus.WithoutScopeInfo(),
		prometheus.WithoutTargetInfo()); err != nil {
		return err
	}
	provider := metric.NewMeterProvider(
		metric.WithReader(m.exporter),
		metric.WithView(
			metric.NewView(
				metric.Instrument{Name: meterLookupRespLatency, Scope: meterScope},
				metric.Stream{
					Aggregation: aggregation.ExplicitBucketHistogram{
						Boundaries: []float64{0, 50, 100, 200, 500, 1000, 5_000, 10_000, 20_000, 30_000, 60_000},
					},
				},
			),
			metric.NewView(
				metric.Instrument{Name: meterLookupRespTTFP, Scope: meterScope},
				metric.Stream{
					Aggregation: aggregation.ExplicitBucketHistogram{
						Boundaries: []float64{0, 50, 100, 200, 300, 400, 500, 1000, 2_000, 5_000, 10_000},
					},
				},
			),
			metric.NewView(
				metric.Instrument{Name: meterLookupRespResultCount, Scope: meterScope},
				metric.Stream{
					Aggregation: aggregation.ExplicitBucketHistogram{
						Boundaries: []float64{0, 1, 2, 5, 10, 15, 20, 25, 50, 100},
					},
				},
			),
		),
	)
	meter := provider.Meter(meterName)

	if m.lookupRequestCounter, err = meter.Int64Counter(
		meterLookupReqCount,
		instrument.WithUnit("1"),
		instrument.WithDescription("The number of lookup requests received."),
	); err != nil {
		return err
	}
	if m.lookupResponseTTFPHistogram, err = meter.Int64Histogram(
		meterLookupRespTTFP,
		instrument.WithUnit("ms"),
		instrument.WithDescription("The elapsed to find the first provider in milliseconds."),
	); err != nil {
		return err
	}
	if m.lookupResponseResultCountHistogram, err = meter.Int64Histogram(
		meterLookupRespResultCount,
		instrument.WithUnit("1"),
		instrument.WithDescription("The number of providers found per lookup."),
	); err != nil {
		return err
	}
	if m.lookupResponseLatencyHistogram, err = meter.Int64Histogram(
		meterLookupRespLatency,
		instrument.WithUnit("ms"),
		instrument.WithDescription("The lookup response latency."),
	); err != nil {
		return err
	}

	m.server.Handler = m.serveMux()
	go func() { _ = m.server.ListenAndServe() }()
	m.server.RegisterOnShutdown(func() {
		// TODO add timeout to exporter shutdown
		if err := m.exporter.Shutdown(context.TODO()); err != nil {
			logger.Errorw("Failed to shut down Prometheus exporter", "err", err)
		}
	})
	logger.Infow("Metric server started", "addr", m.server.Addr)
	return nil
}

func (m *metrics) serveMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if m.c.metricsEnablePprofDebug {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.HandleFunc("/debug/pprof/gc",
			func(w http.ResponseWriter, req *http.Request) {
				runtime.GC()
			},
		)
	}
	return mux
}

func (m *metrics) notifyLookupRequested(ctx context.Context) {
	m.lookupRequestCounter.Add(ctx, 1)
}

func (m *metrics) notifyLookupResponded(ctx context.Context, resultCount int64, timeToFirstResult time.Duration, latency time.Duration) {
	if resultCount > 0 {
		m.lookupResponseTTFPHistogram.Record(ctx, timeToFirstResult.Milliseconds())
	}
	m.lookupResponseResultCountHistogram.Record(ctx, resultCount)
	m.lookupResponseLatencyHistogram.Record(ctx, latency.Milliseconds())
}

func (m *metrics) Shutdown(ctx context.Context) error {
	return m.server.Shutdown(ctx)
}
