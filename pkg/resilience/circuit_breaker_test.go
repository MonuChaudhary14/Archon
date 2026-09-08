package resilience_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MonuChaudhary14/Archon/pkg/resilience"
)

var errDownstreamFailure = errors.New("downstream service outage")

func TestCircuitBreaker_SuccessPassThrough(t *testing.T) {
	cb := resilience.NewCircuitBreaker[string](resilience.Config{
		Name:      "test-service",
		Threshold: 2,
		Timeout:   100 * time.Millisecond,
	})

	val, err := cb.Execute(func() (string, error) {
		return "hello world", nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", val)
	}
	if cb.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_TripToOpenAndFallback(t *testing.T) {
	cb := resilience.NewCircuitBreaker[string](resilience.Config{
		Name:         "test-failing-service",
		Threshold:    2,
		FailureRatio: 0.5,
		Timeout:      150 * time.Millisecond,
	})

	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(func() (string, error) {
			return "", errDownstreamFailure
		})
	}

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen after reaching failure threshold, got %v", cb.State())
	}

	val, err := cb.ExecuteWithFallback(
		func() (string, error) {
			return "fresh data", nil
		},
		func(err error) (string, error) {
			if !resilience.IsCircuitOpenError(err) {
				t.Errorf("expected circuit open error in fallback, got %v", err)
			}
			return "stale cache data", nil
		},
	)

	if err != nil {
		t.Fatalf("expected fallback to succeed, got error %v", err)
	}
	if val != "stale cache data" {
		t.Fatalf("expected 'stale cache data', got '%s'", val)
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := resilience.NewCircuitBreaker[string](resilience.Config{
		Name:         "test-recovery-service",
		Threshold:    2,
		FailureRatio: 0.5,
		Timeout:      50 * time.Millisecond,
	})

	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(func() (string, error) {
			return "", errDownstreamFailure
		})
	}

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	time.Sleep(60 * time.Millisecond)

	val, err := cb.Execute(func() (string, error) {
		return "recovered data", nil
	})

	if err != nil {
		t.Fatalf("expected successful execution after recovery, got %v", err)
	}
	if val != "recovered data" {
		t.Fatalf("expected 'recovered data', got '%s'", val)
	}
	if cb.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed after probe success, got %v", cb.State())
	}
}
