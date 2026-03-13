package apm

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// FieldsFromContext extracts OTel trace/span IDs from ctx and returns
// Datadog-compatible zap fields for log-trace correlation in DD Log Management.
//
// Datadog expects decimal 64-bit IDs under the dd.* keys.
// The OTel TraceID is 128-bit; Datadog uses the lower 64 bits in decimal.
//
// Add dd.service/dd.env/dd.version as static logger fields in main() via
// logger.With(...) — this function handles only the per-request dynamic IDs.
func FieldsFromContext(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	sc := span.SpanContext()

	// Lower 8 bytes of the 128-bit TraceID as decimal uint64.
	tid := sc.TraceID()
	var traceIDLow uint64
	for i := 8; i < 16; i++ {
		traceIDLow = traceIDLow<<8 | uint64(tid[i])
	}

	var spanIDInt uint64
	for _, b := range sc.SpanID() {
		spanIDInt = spanIDInt<<8 | uint64(b)
	}

	return []zap.Field{
		zap.Uint64("dd.trace_id", traceIDLow),
		zap.Uint64("dd.span_id", spanIDInt),
	}
}
