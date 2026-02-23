package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpDuration   metric.Float64Histogram
	httpActiveReqs metric.Int64UpDownCounter
	httpBodySize   metric.Int64Histogram
)

func init() {
	meter := otel.Meter("http")

	var err error
	httpDuration, err = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10),
	)
	if err != nil {
		panic("failed to create httpDuration histogram: " + err.Error())
	}

	httpActiveReqs, err = meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic("failed to create httpActiveReqs counter: " + err.Error())
	}

	httpBodySize, err = meter.Int64Histogram(
		"http.server.request.body.size",
		metric.WithDescription("Size of HTTP server request bodies"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000),
	)
	if err != nil {
		panic("failed to create httpBodySize histogram: " + err.Error())
	}
}

func HTTPMetrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		ctx := c.UserContext()

		routeAttrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Method()),
			attribute.String("http.route", c.Path()),
		}

		httpActiveReqs.Add(ctx, 1, metric.WithAttributes(routeAttrs...))

		contentLength := c.Request().Header.ContentLength()
		if contentLength > 0 {
			httpBodySize.Record(ctx, int64(contentLength), metric.WithAttributes(routeAttrs...))
		}

		err := c.Next()

		elapsed := time.Since(start).Seconds()
		durationAttrs := append(routeAttrs, attribute.Int("http.response.status_code", c.Response().StatusCode()))

		httpDuration.Record(ctx, elapsed, metric.WithAttributes(durationAttrs...))
		httpActiveReqs.Add(ctx, -1, metric.WithAttributes(routeAttrs...))

		return err
	}
}