package metrics

import (
	"reflect"
	"testing"
)

func TestMetricKey(t *testing.T) {
	tests := []struct {
		name   string
		result MetricResult
		want   string
	}{
		{
			name: "metric without labels",
			result: MetricResult{
				Name: "requests_total",
			},
			want: "requests_total{}",
		},
		{
			name: "single label",
			result: MetricResult{
				Name: "requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			want: "requests_total{method=GET}",
		},
		{
			name: "multiple labels are sorted",
			result: MetricResult{
				Name: "requests_total",
				Labels: map[string]string{
					"status": "200",
					"path":   "/users",
					"method": "GET",
				},
			},
			want: "requests_total{method=GET,path=/users,status=200}",
		},
		{
			name: "nil labels",
			result: MetricResult{
				Name:   "requests_total",
				Labels: nil,
			},
			want: "requests_total{}",
		},
		{
			name: "empty label value",
			result: MetricResult{
				Name: "requests_total",
				Labels: map[string]string{
					"method": "",
				},
			},
			want: "requests_total{method=}",
		},
		{
			name: "empty metric name",
			result: MetricResult{
				Name: "",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			want: "{method=GET}",
		},
		{
			name: "special characters in label value",
			result: MetricResult{
				Name: "metric",
				Labels: map[string]string{
					"label": "a=b,c;d",
				},
			},
			want: "metric{label=a=b,c;d}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metricKey(tt.result)

			if got != tt.want {
				t.Fatalf("metricKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMetricKeyIsIndependentOfLabelMapOrder(t *testing.T) {
	first := MetricResult{
		Name: "http_requests_total",
		Type: "counter",
		Labels: map[string]string{
			"method": "GET",
			"status": "200",
			"path":   "/users",
		},
	}

	second := MetricResult{
		Name: "http_requests_total",
		Type: "counter",
		Labels: map[string]string{
			"path":   "/users",
			"status": "200",
			"method": "GET",
		},
	}

	gotFirst := metricKey(first)
	gotSecond := metricKey(second)

	if gotFirst != gotSecond {
		t.Fatalf(
			"metricKey() differs for equivalent label maps: %q != %q",
			gotFirst,
			gotSecond,
		)
	}
}

func TestMetricsStoreSet(t *testing.T) {
	t.Run("stores metric", func(t *testing.T) {
		store := NewMetricsStore()

		metric := MetricResult{
			Name:        "requests_total",
			Description: "Total requests",
			Type:        "counter",
			Value:       42,
			Labels: map[string]string{
				"method": "GET",
			},
		}

		if err := store.Set(metric); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		got := store.GetAll()

		if len(got) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(got))
		}

		key := metricKey(metric)
		gotMetric, ok := got[key]

		if !ok {
			t.Fatalf("metric not found using key %q", key)
		}

		if gotMetric.Name != metric.Name {
			t.Fatalf("Name = %q, want %q", gotMetric.Name, metric.Name)
		}

		if gotMetric.Description != metric.Description {
			t.Fatalf(
				"Description = %q, want %q",
				gotMetric.Description,
				metric.Description,
			)
		}

		if gotMetric.Type != metric.Type {
			t.Fatalf("Type = %q, want %q", gotMetric.Type, metric.Type)
		}

		if gotMetric.Value != metric.Value {
			t.Fatalf("Value = %v, want %v", gotMetric.Value, metric.Value)
		}
	})

	t.Run("overwrites metric with same name and labels", func(t *testing.T) {
		store := NewMetricsStore()

		first := MetricResult{
			Name:        "requests_total",
			Description: "Total requests",
			Type:        "counter",
			Value:       10,
			Labels: map[string]string{
				"method": "GET",
			},
		}

		second := MetricResult{
			Name:        "requests_total",
			Description: "Total requests",
			Type:        "counter",
			Value:       20,
			Labels: map[string]string{
				"method": "GET",
			},
		}

		if err := store.Set(first); err != nil {
			t.Fatalf("failed to store first metric: %v", err)
		}

		if err := store.Set(second); err != nil {
			t.Fatalf("failed to store second metric: %v", err)
		}

		got := store.GetAll()

		if len(got) != 1 {
			t.Fatalf(
				"expected 1 metric after overwrite, got %d",
				len(got),
			)
		}

		key := metricKey(second)

		gotMetric, ok := got[key]

		if !ok {
			t.Fatalf("metric not found using key %q", key)
		}

		if gotMetric.Value != second.Value {
			t.Fatalf(
				"Value = %v, want %v",
				gotMetric.Value,
				second.Value,
			)
		}
	})

	t.Run("same labels in different order overwrite same metric", func(t *testing.T) {
		store := NewMetricsStore()

		first := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 10,
			Labels: map[string]string{
				"method": "GET",
				"status": "200",
			},
		}

		second := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 20,
			Labels: map[string]string{
				"status": "200",
				"method": "GET",
			},
		}

		if err := store.Set(first); err != nil {
			t.Fatalf("failed to store first metric: %v", err)
		}

		if err := store.Set(second); err != nil {
			t.Fatalf("failed to store second metric: %v", err)
		}

		got := store.GetAll()

		if len(got) != 1 {
			t.Fatalf(
				"expected 1 metric, got %d",
				len(got),
			)
		}

		key := metricKey(first)

		gotMetric, ok := got[key]
		if !ok {
			t.Fatalf("metric not found using key %q", key)
		}

		if gotMetric.Value != 20 {
			t.Fatalf("Value = %v, want 20", gotMetric.Value)
		}
	})

	t.Run("different labels create different metrics", func(t *testing.T) {
		store := NewMetricsStore()

		getMetric := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 10,
			Labels: map[string]string{
				"method": "GET",
			},
		}

		postMetric := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 20,
			Labels: map[string]string{
				"method": "POST",
			},
		}

		if err := store.Set(getMetric); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(postMetric); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		got := store.GetAll()

		if len(got) != 2 {
			t.Fatalf("expected 2 metrics, got %d", len(got))
		}

		if _, ok := got[metricKey(getMetric)]; !ok {
			t.Fatal("GET metric not found")
		}

		if _, ok := got[metricKey(postMetric)]; !ok {
			t.Fatal("POST metric not found")
		}
	})

	t.Run("same name but different label sets create different metrics", func(t *testing.T) {
		store := NewMetricsStore()

		withoutLabels := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 10,
		}

		withLabels := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 20,
			Labels: map[string]string{
				"method": "GET",
			},
		}

		store.Set(withoutLabels)
		store.Set(withLabels)

		got := store.GetAll()

		if len(got) != 2 {
			t.Fatalf("expected 2 metrics, got %d", len(got))
		}
	})
}

func TestMetricsStoreGetAll(t *testing.T) {
	t.Run("returns empty map for new store", func(t *testing.T) {
		store := NewMetricsStore()

		got := store.GetAll()

		if got == nil {
			t.Fatal("expected non-nil map")
		}

		if len(got) != 0 {
			t.Fatalf("expected empty map, got %d metrics", len(got))
		}
	})

	t.Run("returns copy of metrics map", func(t *testing.T) {
		store := NewMetricsStore()

		metric := MetricResult{
			Name:  "requests_total",
			Type:  "counter",
			Value: 42,
		}

		if err := store.Set(metric); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		got := store.GetAll()

		delete(got, metricKey(metric))

		gotAgain := store.GetAll()

		if len(gotAgain) != 1 {
			t.Fatalf(
				"modifying GetAll result modified store: got %d metrics",
				len(gotAgain),
			)
		}
	})

	t.Run("returns all stored metrics", func(t *testing.T) {
		store := NewMetricsStore()

		metrics := []MetricResult{
			{
				Name:  "requests_total",
				Type:  "counter",
				Value: 10,
			},
			{
				Name:  "errors_total",
				Type:  "counter",
				Value: 2,
			},
			{
				Name:  "latency",
				Type:  "counter",
				Value: 123.45,
			},
		}

		for _, metric := range metrics {
			if err := store.Set(metric); err != nil {
				t.Fatalf("failed to store metric %q: %v", metric.Name, err)
			}
		}

		got := store.GetAll()

		if len(got) != len(metrics) {
			t.Fatalf(
				"expected %d metrics, got %d",
				len(metrics),
				len(got),
			)
		}

		for _, want := range metrics {
			key := metricKey(want)

			gotMetric, ok := got[key]

			if !ok {
				t.Fatalf("metric %q not found", key)
			}

			if gotMetric.Name != want.Name ||
				gotMetric.Value != want.Value ||
				gotMetric.Description != want.Description ||
				gotMetric.Type != want.Type {
				t.Fatalf(
					"metric %q = %+v, want %+v",
					key,
					gotMetric,
					want,
				)
			}
		}
	})
}

func TestMetricsStoreSetValidMetric(t *testing.T) {
	store := NewMetricsStore()

	metric := MetricResult{
		Name:        "test_metric",
		Description: "Test metric",
		Type:        "gauge",
		Value:       42,
	}

	err := store.Set(metric)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	metrics := store.GetAll()

	key := metricKey(metric)

	storedMetric, ok := metrics[key]

	if !ok {
		t.Fatalf("expected metric %q to be stored", key)
	}

	if !reflect.DeepEqual(storedMetric, metric) {
		t.Fatalf("expected stored metric %+v, got %+v", metric, storedMetric)
	}
}

func TestMetricsStoreSetInvalidMetric(t *testing.T) {
	store := NewMetricsStore()

	metric := MetricResult{
		Name:        "test_metric",
		Description: "Test metric",
		Type:        "gauge",
		Value:       42,
		Labels: map[string]string{
			"invalid-label": "value",
		},
	}

	err := store.Set(metric)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	metrics := store.GetAll()

	key := metricKey(metric)

	if _, ok := metrics[key]; ok {
		t.Fatalf("expected metric %q not to be stored", key)
	}
}

func BenchmarkMetricsStoreSet(b *testing.B) {
	store := NewMetricsStore()

	metric := MetricResult{
		Name:        "cpu_usage_percent",
		Description: "CPU usage percentage",
		Type:        "gauge",
		Value:       42.5,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := store.Set(metric); err != nil {
			b.Fatal(err)
		}
	}
}
