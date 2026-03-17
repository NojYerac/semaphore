package data

import (
	"context"
	"time"

	"github.com/nojyerac/go-lib/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	initial                bool
	flagEvaluationDuration metric.Int64Histogram
	flagEvaluationCount    metric.Int64Counter
	flagCountGauge         metric.Int64ObservableGauge
	meter                  = metrics.MeterForPackage()
)

var _ DataEngine = (*meteredDataEngine)(nil)

func NewMeteredDataEngine(source Source, engine Engine) (DataEngine, error) {
	de := &meteredDataEngine{
		DataEngine: NewDataEngine(source, engine),
	}
	if err := initMetrics(de); err != nil {
		return nil, err
	}
	return de, nil
}

type meteredDataEngine struct {
	DataEngine
	reg metric.Registration
}

func (m *meteredDataEngine) EvaluateFlag(ctx context.Context, flagID, userID string, groupIDs []string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		attrs := []attribute.KeyValue{
			attribute.String("flag_id", flagID),
		}
		flagEvaluationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		flagEvaluationCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()
	return m.DataEngine.EvaluateFlag(ctx, flagID, userID, groupIDs)
}

func initMetrics(de *meteredDataEngine) error {
	if !initial {
		initial = true
		var err error
		flagEvaluationDuration, err = meter.Int64Histogram(
			"flag_evaluation_duration",
			metric.WithDescription("duration of flag evaluations"),
			metric.WithUnit("ms"),
			metric.WithExplicitBucketBoundaries(1, 2, 4, 8, 16, 32, 64, 128, 256, 512),
		)
		if err != nil {
			return err
		}
		flagEvaluationCount, err = meter.Int64Counter(
			"flag_evaluation_count",
			metric.WithDescription("count of flag evaluations"),
		)
		if err != nil {
			return err
		}
		flagCountGauge, err = meter.Int64ObservableGauge(
			"flags_total",
			metric.WithDescription("count of flags by enabled/disabled status"),
		)
		if err != nil {
			return err
		}
	}

	callBack := func(ctx context.Context, o metric.Observer) error {
		flags, err := de.GetFlags(ctx)
		if err != nil {
			return err
		}
		var enabledCount, disabledCount int
		for _, flag := range flags {
			if flag.Enabled {
				enabledCount++
			} else {
				disabledCount++
			}
		}
		o.ObserveInt64(flagCountGauge, int64(enabledCount), metric.WithAttributes(
			attribute.Bool("flag_enabled", true),
		))
		o.ObserveInt64(flagCountGauge, int64(disabledCount), metric.WithAttributes(
			attribute.Bool("flag_enabled", false),
		))
		return nil
	}
	reg, err := meter.RegisterCallback(callBack, flagCountGauge)
	if err != nil {
		return err
	}
	de.reg = reg
	return nil
}
