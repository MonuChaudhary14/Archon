package resilience

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

var (
	ErrCircuitOpen = gobreaker.ErrOpenState
	ErrTooManyRequests = gobreaker.ErrTooManyRequests
)

type State = gobreaker.State

const (
	StateClosed   = gobreaker.StateClosed
	StateHalfOpen = gobreaker.StateHalfOpen
	StateOpen     = gobreaker.StateOpen
)

type Config struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	Threshold     uint32
	FailureRatio  float64
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(err error) bool
}

type CircuitBreaker[T any] struct {
	cb *gobreaker.CircuitBreaker[T]
}

func NewCircuitBreaker[T any](cfg Config) *CircuitBreaker[T] {
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 1
	}
	if cfg.Interval == 0 {
		cfg.Interval = 10 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = 5
	}
	if cfg.FailureRatio == 0 {
		cfg.FailureRatio = 0.5
	}

	readyToTrip := func(counts gobreaker.Counts) bool {
		if counts.Requests < cfg.Threshold {
			return false
		}
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return failureRatio >= cfg.FailureRatio
	}

	st := gobreaker.Settings{
		Name:          cfg.Name,
		MaxRequests:   cfg.MaxRequests,
		Interval:      cfg.Interval,
		Timeout:       cfg.Timeout,
		ReadyToTrip:   readyToTrip,
		OnStateChange: cfg.OnStateChange,
		IsSuccessful:  cfg.IsSuccessful,
	}

	return &CircuitBreaker[T]{
		cb: gobreaker.NewCircuitBreaker[T](st),
	}
}

func (c *CircuitBreaker[T]) Execute(req func() (T, error)) (T, error) {
	return c.cb.Execute(req)
}

func (c *CircuitBreaker[T]) ExecuteWithFallback(req func() (T, error), fallback func(err error) (T, error)) (T, error) {
	result, err := c.cb.Execute(req)
	if err != nil {
		if fallback != nil {
			return fallback(err)
		}
		return result, err
	}
	return result, nil
}

func (c *CircuitBreaker[T]) State() State {
	return c.cb.State()
}

func (c *CircuitBreaker[T]) Counts() gobreaker.Counts {
	return c.cb.Counts()
}

func IsCircuitOpenError(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)
}
