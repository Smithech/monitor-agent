package monitor

import (
	"context"
	"fmt"
	"time"

	"monitor-agent/internal/collector"
	"monitor-agent/internal/metrics"
)

func RunMonitor(
	ctx context.Context,
	name string,
	results chan metrics.MetricResult,
	statusStore *CollectorStatusStore,
	interval time.Duration,
	retryInterval time.Duration,
	collect func() ([]metrics.MetricResult, error),
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		metrics, err := collect()

		if err != nil {
			collectorErr := newCollectorError(err)

			if err := statusStore.SetState(name, collectorErr.State); err != nil {
				fmt.Printf("%s status error: %v\n", name, err)
			}

			fmt.Printf("%s monitor error: %v\n", name, err)

			select {
			case <-time.After(retryInterval):
			case <-ctx.Done():
				return
			}

			continue
		}

		if err := statusStore.SetState(name, CollectorAvailable); err != nil {
			fmt.Printf("%s status error: %v\n", name, err)
		}

		for _, metric := range metrics {
			select {
			case results <- metric:
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func MonitorCPU(
	ctx context.Context,
	results chan metrics.MetricResult,
	statusStore *CollectorStatusStore,
	interval time.Duration,
	retryInterval time.Duration,
	reader collector.CPUStatsReader,
) {
	var previous collector.CPUStats

	for {
		var err error
		previous, err = reader()

		if err == nil {
			if err := statusStore.SetState("cpu", CollectorAvailable); err != nil {
				fmt.Printf("cpu status error: %v\n", err)
			}
			break
		}

		collectorErr := newCollectorError(err)

		if err := statusStore.SetState("cpu", collectorErr.State); err != nil {
			fmt.Printf("cpu status error: %v\n", err)
		}

		fmt.Printf("cpu monitor error: %v\n", err)

		select {
		case <-time.After(retryInterval):
		case <-ctx.Done():
			return
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}

		current, metric, err := collector.CollectCPU(previous, reader)

		if err != nil {
			collectorErr := newCollectorError(err)

			if err := statusStore.SetState("cpu", collectorErr.State); err != nil {
				fmt.Printf("cpu status error: %v\n", err)
			}

			fmt.Printf("cpu monitor error: %v\n", err)

			previous = current

			continue
		}

		if err := statusStore.SetState("cpu", CollectorAvailable); err != nil {
			fmt.Printf("cpu status error: %v\n", err)
		}

		previous = current

		select {
		case results <- metric:
		case <-ctx.Done():
			return
		}
	}
}
