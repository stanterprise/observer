package consumer

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// consumerMetrics holds all OTel instruments used by NATSConsumer.
// All fields are always initialised (to no-op instruments when telemetry is
// not configured) so nil checks are never required at call sites.
type consumerMetrics struct {
	// Per-event-type counters
	eventsProcessed metric.Int64Counter
	eventsFailed    metric.Int64Counter
	eventsDLQ       metric.Int64Counter
	eventsDeferred  metric.Int64Counter

	// Business-level run / test counters
	runsStarted    metric.Int64Counter
	runsCompleted  metric.Int64Counter
	testsCompleted metric.Int64Counter

	// Processing latency (seconds) labelled by event_type
	processingDuration metric.Float64Histogram

	// Batch sizes returned by each NATS Fetch call
	batchSize metric.Int64Histogram
}

// initConsumerMetrics creates all instruments from the supplied Meter and
// registers an observable gauge for the deferred queue depth.
// It returns a registration handle whose Unregister method must be called
// when the consumer is closed.
func initConsumerMetrics(meter metric.Meter, queueDepthFn func() int64) (*consumerMetrics, metric.Registration, error) {
	m := &consumerMetrics{}
	var err error

	m.eventsProcessed, err = meter.Int64Counter(
		"observer.processor.events.processed",
		metric.WithDescription("Total NATS events successfully processed, labelled by event_type"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.eventsFailed, err = meter.Int64Counter(
		"observer.processor.events.failed",
		metric.WithDescription("Total NATS events that failed processing, labelled by event_type"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.eventsDLQ, err = meter.Int64Counter(
		"observer.processor.events.dlq",
		metric.WithDescription("Total NATS events forwarded to the dead-letter queue, labelled by event_type"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.eventsDeferred, err = meter.Int64Counter(
		"observer.processor.events.deferred",
		metric.WithDescription("Total step events deferred to the in-memory retry queue, labelled by event_type"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.runsStarted, err = meter.Int64Counter(
		"observer.processor.runs.started",
		metric.WithDescription("Total test runs started (RunStart events processed)"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.runsCompleted, err = meter.Int64Counter(
		"observer.processor.runs.completed",
		metric.WithDescription("Total test runs completed (RunEnd events processed), labelled by status"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.testsCompleted, err = meter.Int64Counter(
		"observer.processor.tests.completed",
		metric.WithDescription("Total test cases completed (TestEnd events processed), labelled by status"),
	)
	if err != nil {
		return nil, nil, err
	}

	m.processingDuration, err = meter.Float64Histogram(
		"observer.processor.event.processing.duration",
		metric.WithDescription("Wall-clock time in seconds to process a single NATS message, labelled by event_type"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.001, 0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1.0, 2.5, 5.0,
		),
	)
	if err != nil {
		return nil, nil, err
	}

	m.batchSize, err = meter.Int64Histogram(
		"observer.processor.batch.size",
		metric.WithDescription("Number of messages returned per NATS Fetch call"),
		metric.WithExplicitBucketBoundaries(0, 1, 5, 10, 25, 50, 100),
	)
	if err != nil {
		return nil, nil, err
	}

	// Observable gauge: total pending deferred step events across all queues.
	depthGauge, err := meter.Int64ObservableGauge(
		"observer.processor.deferred_queue.depth",
		metric.WithDescription("Total orphan step events currently held in the deferred retry queue"),
	)
	if err != nil {
		return nil, nil, err
	}

	reg, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(depthGauge, queueDepthFn())
		return nil
	}, depthGauge)
	if err != nil {
		return nil, nil, err
	}

	return m, reg, nil
}

// eventAttr returns an OTel attribute for labelling counters / histograms by
// event type.
func eventAttr(eventType string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("event_type", eventType))
}

// statusAttr returns an OTel attribute for labelling counters by status.
func statusAttr(status string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("status", status))
}
