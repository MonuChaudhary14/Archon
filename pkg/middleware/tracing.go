package middleware

import (
	"fmt"

	"github.com/MonuChaudhary14/Archon/pkg/telemetry"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware() gin.HandlerFunc {
	propagator := otel.GetTextMapPropagator()
	tracer := telemetry.Tracer()

	return func(c *gin.Context) {
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := c.FullPath()
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s", c.Request.Method)
		}

		opts := []trace.SpanStartOption{
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPTargetKey.String(c.Request.URL.Path),
				semconv.HTTPRouteKey.String(c.FullPath()),
				semconv.NetHostNameKey.String(c.Request.Host),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		}

		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		if span.SpanContext().IsValid() {
			c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
		}

		c.Next()

		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(statusCode))

		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		} else if statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		if userID, exists := c.Get("userID"); exists {
			span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
		}
	}
}
