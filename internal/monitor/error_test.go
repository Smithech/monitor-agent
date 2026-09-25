package monitor

import (
	"errors"
	"testing"

	"monitor-agent/internal/collector"
)

func TestNewCollectorError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		got := newCollectorError(nil)

		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("unavailable collector", func(t *testing.T) {
		err := errors.New("read failed")

		got := newCollectorError(err)

		if got == nil {
			t.Fatal("expected collector error")
		}

		if got.State != CollectorUnavailable {
			t.Fatalf(
				"expected state %q, got %q",
				CollectorUnavailable,
				got.State,
			)
		}

		if !errors.Is(got.Err, err) {
			t.Fatal("expected original error to be preserved")
		}
	})

	t.Run("unsupported temperature collector", func(t *testing.T) {
		got := newCollectorError(collector.ErrTemperatureNotSupported)

		if got == nil {
			t.Fatal("expected collector error")
		}

		if got.State != CollectorNotSupported {
			t.Fatalf(
				"expected state %q, got %q",
				CollectorNotSupported,
				got.State,
			)
		}

		if !errors.Is(got.Err, collector.ErrTemperatureNotSupported) {
			t.Fatal("expected original error to be preserved")
		}
	})
}
