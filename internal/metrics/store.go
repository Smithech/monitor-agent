package metrics

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type MetricsStore struct {
	metrics map[string]MetricResult
	mu      sync.RWMutex
}

func NewMetricsStore() *MetricsStore {
	return &MetricsStore{
		metrics: make(map[string]MetricResult),
	}
}

func (m *MetricsStore) Set(result MetricResult) error {
	if err := validateMetric(result); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey(result)
	m.metrics[key] = result

	return nil
}

func (m *MetricsStore) GetAll() map[string]MetricResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]MetricResult, len(m.metrics))

	maps.Copy(metrics, m.metrics)

	return metrics
}

// codigo duplicado con metricKey()
func formatMetric(result MetricResult) string {
	metric := result.Name

	if len(result.Labels) > 0 {
		keys := make([]string, 0, len(result.Labels))

		for key := range result.Labels {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		labels := make([]string, 0, len(keys))

		for _, key := range keys {
			labels = append(
				labels,
				fmt.Sprintf(`%s="%s"`, key, result.Labels[key]),
			)
		}

		metric += "{" + strings.Join(labels, ",") + "}"
	}

	value := strconv.FormatFloat(result.Value, 'f', -1, 64)

	return fmt.Sprintf("%s %s", metric, value)
}

func metricKey(result MetricResult) string {
	keys := make([]string, 0, len(result.Labels))

	for key := range result.Labels {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	labels := make([]string, 0, len(keys))

	for _, key := range keys {
		labels = append(labels, fmt.Sprintf("%s=%s", key, result.Labels[key]))
	}

	return fmt.Sprintf("%s{%s}", result.Name, strings.Join(labels, ","))
}
