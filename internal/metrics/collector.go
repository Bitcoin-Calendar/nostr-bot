package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

// Collector stores metrics about bot operation.
// It includes fields for both general operation and NIP-68 specific events.
type Collector struct {
	Registry             *prometheus.Registry
	Kind1EventsPosted    prometheus.Counter
	Kind1EventsFailed    prometheus.Counter
	Kind20EventsPosted   prometheus.Counter
	Kind20EventsFailed   prometheus.Counter
	Kind20EventsSkipped  prometheus.Counter
	ImageValidationFails prometheus.Counter
	EventsSkipped        prometheus.Counter
	RelaySuccesses       *prometheus.CounterVec
	RelayFailures        *prometheus.CounterVec
	RelaySuccessTimes    *prometheus.HistogramVec
}

// NewCollector initializes a new MetricsCollector.
func NewCollector() *Collector {
	reg := prometheus.NewRegistry()
	c := &Collector{
		Registry: reg,
		Kind1EventsPosted: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_kind1_events_posted_total",
			Help: "The total number of kind 1 events successfully posted.",
		}),
		Kind1EventsFailed: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_kind1_events_failed_total",
			Help: "The total number of kind 1 events that failed to post.",
		}),
		Kind20EventsPosted: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_kind20_events_posted_total",
			Help: "The total number of kind 20 events successfully posted.",
		}),
		Kind20EventsFailed: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_kind20_events_failed_total",
			Help: "The total number of kind 20 events that failed to post.",
		}),
		Kind20EventsSkipped: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_kind20_events_skipped_total",
			Help: "The total number of kind 20 events skipped.",
		}),
		ImageValidationFails: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_image_validation_fails_total",
			Help: "The total number of image validation fails.",
		}),
		EventsSkipped: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "nostr_bot_events_skipped_total",
			Help: "The total number of events skipped because they were not for today.",
		}),
		RelaySuccesses: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "nostr_bot_relay_successes_total",
			Help: "The total number of successful publishes to a relay.",
		}, []string{"relay_url"}),
		RelayFailures: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "nostr_bot_relay_failures_total",
			Help: "The total number of failed publishes to a relay.",
		}, []string{"relay_url"}),
		RelaySuccessTimes: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nostr_bot_relay_success_duration_seconds",
			Help:    "The duration of successful publishes to a relay.",
			Buckets: prometheus.DefBuckets,
		}, []string{"relay_url"}),
	}
	return c
}

// RecordRelaySuccess records a successful relay publish.
func (mc *Collector) RecordRelaySuccess(relayURL string, duration time.Duration) {
	mc.RelaySuccesses.WithLabelValues(relayURL).Inc()
	mc.RelaySuccessTimes.WithLabelValues(relayURL).Observe(duration.Seconds())
}

// RecordRelayFailure records a failed relay publish.
func (mc *Collector) RecordRelayFailure(relayURL string) {
	mc.RelayFailures.WithLabelValues(relayURL).Inc()
}

// LogSummary logs a summary of collected metrics using the global logger.
// This will need to be updated to show the new NIP-68 fields.
func (mc *Collector) LogSummary() {
	// This function can be simplified or removed, as metrics are now exposed via Prometheus.
	// For now, it logs the current state of the counters.
	log.Info().Msg("Metrics are now exposed via the /metrics endpoint. This log is for debugging.")
}

// ExportMetrics saves the collected metrics to a JSON file.
func (mc *Collector) ExportMetrics(filePath string) error {
	// This function is kept for backward compatibility, but Prometheus is the primary metrics store.
	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metrics JSON to file %s: %w", filePath, err)
	}
	return nil
}
