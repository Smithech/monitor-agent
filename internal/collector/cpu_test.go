package collector

import (
	"errors"
	"reflect"
	"testing"

	"monitor-agent/internal/metrics"
)

func TestCPUUsageToMetric(t *testing.T) {
	metric := cpuUsageToMetric(42.5)

	if metric.Name != "cpu_usage_percent" {
		t.Errorf("unexpected metric name: %s", metric.Name)
	}

	if metric.Description != "CPU usage percentage" {
		t.Errorf("unexpected description: %s", metric.Description)
	}

	if metric.Type != "gauge" {
		t.Errorf("unexpected metric type: %s", metric.Type)
	}

	if metric.Value != 42.5 {
		t.Errorf("unexpected value: %f", metric.Value)
	}
}

func TestCollectCPU(t *testing.T) {
	previous := CPUStats{
		User:   100,
		System: 100,
		Idle:   800,
	}

	current := CPUStats{
		User:   150,
		System: 120,
		Idle:   830,
	}

	reader := func() (CPUStats, error) {
		return current, nil
	}

	gotCurrent, metric, err := CollectCPU(previous, reader)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotCurrent != current {
		t.Errorf("unexpected current stats: %+v", gotCurrent)
	}

	if metric.Name != "cpu_usage_percent" {
		t.Errorf("unexpected metric name: %s", metric.Name)
	}

	if metric.Value != 70 {
		t.Errorf("unexpected metric value: %f", metric.Value)
	}
}

func TestCollectCPUReaderError(t *testing.T) {
	previous := CPUStats{
		User:   100,
		System: 100,
		Idle:   800,
	}

	expectedErr := errors.New("reader error")

	reader := func() (CPUStats, error) {
		return CPUStats{}, expectedErr
	}

	gotCurrent, metric, err := CollectCPU(previous, reader)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if gotCurrent != (CPUStats{}) {
		t.Errorf("expected zero current stats, got %+v", gotCurrent)
	}

	if !reflect.DeepEqual(metric, metrics.MetricResult{}) {
		t.Errorf("expected zero metric, got %+v", metric)
	}
}

func TestCollectCPUCalculateCPUUsageError(t *testing.T) {
	previous := CPUStats{
		User:   100,
		System: 100,
		Idle:   800,
	}

	// User decreases from 100 to 50, causing calculateCPUUsage
	// -> calculateCPUDeltas -> calculateDelta to return an error.
	current := CPUStats{
		User:   50,
		System: 120,
		Idle:   830,
	}

	reader := func() (CPUStats, error) {
		return current, nil
	}

	gotCurrent, metric, err := CollectCPU(previous, reader)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errInvalidDelta) {
		t.Errorf("expected errInvalidDelta, got %v", err)
	}

	if gotCurrent != current {
		t.Errorf("expected current stats %+v, got %+v", current, gotCurrent)
	}

	if !reflect.DeepEqual(metric, metrics.MetricResult{}) {
		t.Errorf("expected zero metric, got %+v", metric)
	}
}

func BenchmarkCalculateCPUUsage(b *testing.B) {
	previous := CPUStats{
		User:    100000,
		Nice:    20000,
		System:  50000,
		Idle:    300000,
		IOWait:  10000,
		IRQ:     5000,
		SoftIRQ: 8000,
		Steal:   1000,
	}

	current := CPUStats{
		User:    100100,
		Nice:    20020,
		System:  50040,
		Idle:    300200,
		IOWait:  10010,
		IRQ:     5005,
		SoftIRQ: 8008,
		Steal:   1001,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := calculateCPUUsage(previous, current)
		if err != nil {
			b.Fatal(err)
		}
	}
}
