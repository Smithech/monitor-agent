package monitor

import (
	"errors"

	"monitor-agent/internal/collector"
)

type CollectorError struct {
	State CollectorState
	Err   error
}

func newCollectorError(err error) *CollectorError {
	if err == nil {
		return nil
	}

	state := CollectorUnavailable

	if errors.Is(err, collector.ErrTemperatureNotSupported) {
		state = CollectorNotSupported
	}

	return &CollectorError{
		State: state,
		Err:   err,
	}
}
