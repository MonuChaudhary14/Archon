package telemetry

import (
	"context"
	"testing"
)

func TestInitTracer(t *testing.T) {
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "localhost:4317")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}

	tracer := Tracer()
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	_, span := tracer.Start(ctx, "test-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Logf("shutdown notice: %v", err)
	}
}
