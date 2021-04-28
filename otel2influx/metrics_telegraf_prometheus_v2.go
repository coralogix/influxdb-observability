package otel2influx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/influxdata/influxdb-observability/common"
	otlpcommon "github.com/influxdata/influxdb-observability/otlp/common/v1"
	otlpmetrics "github.com/influxdata/influxdb-observability/otlp/metrics/v1"
	otlpresource "github.com/influxdata/influxdb-observability/otlp/resource/v1"
)

type metricWriterTelegrafPrometheusV2 struct {
	logger common.Logger
}

func (c *metricWriterTelegrafPrometheusV2) writeMetric(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, metric *otlpmetrics.Metric, w InfluxWriter) error {
	// Ignore metric.Description() and metric.Unit() .
	switch metricData := metric.Data.(type) {
	case *otlpmetrics.Metric_DoubleGauge:
		return c.writeDoubleGauge(ctx, resource, instrumentationLibrary, metric.Name, metricData.DoubleGauge, w)
	case *otlpmetrics.Metric_IntGauge:
		return c.writeIntGauge(ctx, resource, instrumentationLibrary, metric.Name, metricData.IntGauge, w)
	case *otlpmetrics.Metric_DoubleSum:
		return c.writeDoubleSum(ctx, resource, instrumentationLibrary, metric.Name, metricData.DoubleSum, w)
	case *otlpmetrics.Metric_IntSum:
		return c.writeIntSum(ctx, resource, instrumentationLibrary, metric.Name, metricData.IntSum, w)
	case *otlpmetrics.Metric_DoubleHistogram:
		return c.writeDoubleHistogram(ctx, resource, instrumentationLibrary, metric.Name, metricData.DoubleHistogram, w)
	case *otlpmetrics.Metric_IntHistogram:
		return c.writeIntHistogram(ctx, resource, instrumentationLibrary, metric.Name, metricData.IntHistogram, w)
	case *otlpmetrics.Metric_DoubleSummary:
		return c.writeDoubleSummary(ctx, resource, instrumentationLibrary, metric.Name, metricData.DoubleSummary, w)
	default:
		return fmt.Errorf("unknown metric type %T", metric.Data)
	}
}

func (c *metricWriterTelegrafPrometheusV2) initMetricTagsAndTimestamp(resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, timeUnixNano uint64, labels []*otlpcommon.StringKeyValue) (tags map[string]string, fields map[string]interface{}, ts time.Time, err error) {
	ts = time.Unix(0, int64(timeUnixNano))
	if ts.IsZero() {
		err = errors.New("metric has no timestamp")
		return
	}

	tags = make(map[string]string)
	fields = make(map[string]interface{})

	for _, label := range labels {
		if k, v := label.Key, label.Value; k == "" {
			c.logger.Debug("metric label key is empty")
		} else {
			tags[k] = v
		}
	}

	tags = resourceToTags(c.logger, resource, tags)
	tags = instrumentationLibraryToTags(instrumentationLibrary, tags)

	return
}

func (c *metricWriterTelegrafPrometheusV2) writeDoubleGauge(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, gauge *otlpmetrics.DoubleGauge, w InfluxWriter) error {
	for _, dataPoint := range gauge.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		fields[measurement] = dataPoint.Value

		if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, fields, ts); err != nil {
			return fmt.Errorf("failed to write point for gauge: %w", err)
		}
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeIntGauge(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, gauge *otlpmetrics.IntGauge, w InfluxWriter) error {
	for _, dataPoint := range gauge.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		fields[measurement] = float64(dataPoint.Value)

		if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, fields, ts); err != nil {
			return fmt.Errorf("failed to write point for gauge: %w", err)
		}
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeDoubleSum(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, sum *otlpmetrics.DoubleSum, w InfluxWriter) error {
	if sum.AggregationTemporality != otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE {
		return fmt.Errorf("unsupported sum aggregation temporality %q", sum.AggregationTemporality)
	}
	if !sum.IsMonotonic {
		return fmt.Errorf("unsupported non-monotonic sum '%s'", measurement)
	}

	for _, dataPoint := range sum.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		fields[measurement] = dataPoint.Value

		if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, fields, ts); err != nil {
			return fmt.Errorf("failed to write point for sum: %w", err)
		}
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeIntSum(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, sum *otlpmetrics.IntSum, w InfluxWriter) error {
	if sum.AggregationTemporality != otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE {
		return fmt.Errorf("unsupported sum aggregation temporality %q", sum.AggregationTemporality)
	}
	if !sum.IsMonotonic {
		return fmt.Errorf("unsupported non-monotonic sum '%s'", measurement)
	}

	for _, dataPoint := range sum.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		fields[measurement] = float64(dataPoint.Value)

		if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, fields, ts); err != nil {
			return fmt.Errorf("failed to write point for sum: %w", err)
		}
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeDoubleHistogram(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, histogram *otlpmetrics.DoubleHistogram, w InfluxWriter) error {
	if histogram.AggregationTemporality != otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE {
		return fmt.Errorf("unsupported histogram aggregation temporality %q", histogram.AggregationTemporality)
	}

	for _, dataPoint := range histogram.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		{
			f := make(map[string]interface{}, len(fields)+2)
			for k, v := range fields {
				f[k] = v
			}

			f[measurement+common.MetricHistogramCountSuffix] = float64(dataPoint.Count)
			f[measurement+common.MetricHistogramSumSuffix] = dataPoint.Sum

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, f, ts); err != nil {
				return fmt.Errorf("failed to write point for histogram: %w", err)
			}
		}

		bucketCounts, explicitBounds := dataPoint.BucketCounts, dataPoint.ExplicitBounds
		if len(bucketCounts) > 0 && len(bucketCounts) != len(explicitBounds)+1 {
			return fmt.Errorf("invalid metric histogram bucket counts qty %d vs explicit bounds qty %d", len(bucketCounts), len(explicitBounds))
		}

		for i, explicitBound := range explicitBounds {
			t := make(map[string]string, len(tags)+1)
			for k, v := range tags {
				t[k] = v
			}
			f := make(map[string]interface{}, len(fields)+1)
			for k, v := range fields {
				f[k] = v
			}

			boundTagValue := strconv.FormatFloat(explicitBound, 'f', -1, 64)
			t[common.MetricHistogramBoundKeyV2] = boundTagValue
			f[measurement+common.MetricHistogramBucketSuffix] = float64(bucketCounts[i])

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, t, f, ts); err != nil {
				return fmt.Errorf("failed to write point for histogram: %w", err)
			}
		} // Skip last bucket count - infinity not used in this schema
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeIntHistogram(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, histogram *otlpmetrics.IntHistogram, w InfluxWriter) error {
	if histogram.AggregationTemporality != otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE {
		return fmt.Errorf("unsupported histogram aggregation temporality %q", histogram.AggregationTemporality)
	}

	for _, dataPoint := range histogram.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		{
			f := make(map[string]interface{}, len(fields)+2)
			for k, v := range fields {
				f[k] = v
			}

			f[measurement+common.MetricHistogramCountSuffix] = float64(dataPoint.Count)
			f[measurement+common.MetricHistogramSumSuffix] = float64(dataPoint.Sum)

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, f, ts); err != nil {
				return fmt.Errorf("failed to write point for histogram: %w", err)
			}
		}

		bucketCounts, explicitBounds := dataPoint.BucketCounts, dataPoint.ExplicitBounds
		if len(bucketCounts) > 0 && len(bucketCounts) != len(explicitBounds)+1 {
			return fmt.Errorf("invalid metric histogram bucket counts qty %d vs explicit bounds qty %d", len(bucketCounts), len(explicitBounds))
		}

		for i, explicitBound := range explicitBounds {
			t := make(map[string]string, len(tags)+1)
			for k, v := range tags {
				t[k] = v
			}
			f := make(map[string]interface{}, len(fields)+1)
			for k, v := range fields {
				f[k] = v
			}

			boundTagValue := strconv.FormatFloat(explicitBound, 'f', -1, 64)
			t[common.MetricHistogramBoundKeyV2] = boundTagValue
			f[measurement+common.MetricHistogramBucketSuffix] = float64(bucketCounts[i])

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, t, f, ts); err != nil {
				return fmt.Errorf("failed to write point for histogram: %w", err)
			}
		} // Skip last bucket count - infinity not used in this schema
	}

	return nil
}

func (c *metricWriterTelegrafPrometheusV2) writeDoubleSummary(ctx context.Context, resource *otlpresource.Resource, instrumentationLibrary *otlpcommon.InstrumentationLibrary, measurement string, summary *otlpmetrics.DoubleSummary, w InfluxWriter) error {
	for _, dataPoint := range summary.DataPoints {
		tags, fields, ts, err := c.initMetricTagsAndTimestamp(resource, instrumentationLibrary, dataPoint.TimeUnixNano, dataPoint.Labels)
		if err != nil {
			return err
		}

		{
			f := make(map[string]interface{}, len(fields)+2)
			for k, v := range fields {
				f[k] = v
			}

			f[measurement+common.MetricSummaryCountSuffix] = float64(dataPoint.Count)
			f[measurement+common.MetricSummarySumSuffix] = dataPoint.Sum

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, tags, f, ts); err != nil {
				return fmt.Errorf("failed to write point for summary: %w", err)
			}
		}

		for _, valueAtQuantile := range dataPoint.QuantileValues {
			t := make(map[string]string, len(tags)+1)
			for k, v := range tags {
				t[k] = v
			}
			f := make(map[string]interface{}, len(fields)+1)
			for k, v := range fields {
				f[k] = v
			}

			quantileTagValue := strconv.FormatFloat(valueAtQuantile.Quantile, 'f', -1, 64)
			t[common.MetricSummaryQuantileKeyV2] = quantileTagValue
			f[measurement] = float64(valueAtQuantile.Value)

			if err = w.WritePoint(ctx, common.MeasurementPrometheus, t, f, ts); err != nil {
				return fmt.Errorf("failed to write point for summary: %w", err)
			}
		}
	}

	return nil
}
