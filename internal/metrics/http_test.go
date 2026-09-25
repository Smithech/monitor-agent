package metrics

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupMetrics(t *testing.T) {
	t.Run("groups metrics by name", func(t *testing.T) {
		metrics := map[string]MetricResult{
			"one": {
				Name:  "requests_total",
				Value: 1,
			},
			"two": {
				Name:  "requests_total",
				Value: 2,
			},
			"three": {
				Name:  "errors_total",
				Value: 3,
			},
		}

		got := groupMetrics(metrics)

		if len(got) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(got))
		}

		requests := got["requests_total"]
		if len(requests) != 2 {
			t.Fatalf("expected 2 requests_total metrics, got %d", len(requests))
		}

		errors := got["errors_total"]
		if len(errors) != 1 {
			t.Fatalf("expected 1 errors_total metric, got %d", len(errors))
		}
	})

	t.Run("empty input returns empty map", func(t *testing.T) {
		got := groupMetrics(map[string]MetricResult{})

		if got == nil {
			t.Fatal("expected non-nil map")
		}

		if len(got) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(got))
		}
	})
}

func TestLabelsKey(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "nil labels",
			labels: nil,
			want:   "",
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   "",
		},
		{
			name: "single label",
			labels: map[string]string{
				"method": "GET",
			},
			want: "method=GET;",
		},
		{
			name: "labels are sorted by key",
			labels: map[string]string{
				"status": "200",
				"method": "GET",
				"path":   "/users",
			},
			want: "method=GET;path=/users;status=200;",
		},
		{
			name: "empty label value",
			labels: map[string]string{
				"method": "",
			},
			want: "method=;",
		},
		{
			name: "special characters in values",
			labels: map[string]string{
				"a": "x=y;z",
			},
			want: "a=x=y;z;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := labelsKey(tt.labels)

			if got != tt.want {
				t.Fatalf("labelsKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMetric(t *testing.T) {
	tests := []struct {
		name   string
		result MetricResult
		want   string
	}{
		{
			name: "metric without labels",
			result: MetricResult{
				Name:  "requests_total",
				Value: 42,
			},
			want: "requests_total 42",
		},
		{
			name: "metric with one label",
			result: MetricResult{
				Name: "requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
				Value: 42,
			},
			want: `requests_total{method="GET"} 42`,
		},
		{
			name: "metric with multiple labels sorted by key",
			result: MetricResult{
				Name: "requests_total",
				Labels: map[string]string{
					"status": "200",
					"path":   "/users",
					"method": "GET",
				},
				Value: 42,
			},
			want: `requests_total{method="GET",path="/users",status="200"} 42`,
		},
		{
			name: "zero value",
			result: MetricResult{
				Name:  "requests_total",
				Value: 0,
			},
			want: "requests_total 0",
		},
		{
			name: "negative value",
			result: MetricResult{
				Name:  "temperature",
				Value: -12.5,
			},
			want: "temperature -12.5",
		},
		{
			name: "decimal value",
			result: MetricResult{
				Name:  "ratio",
				Value: 123.456789,
			},
			want: "ratio 123.456789",
		},
		{
			name: "positive infinity",
			result: MetricResult{
				Name:  "value",
				Value: math.Inf(1),
			},
			want: "value +Inf",
		},
		{
			name: "negative infinity",
			result: MetricResult{
				Name:  "value",
				Value: math.Inf(-1),
			},
			want: "value -Inf",
		},
		{
			name: "NaN",
			result: MetricResult{
				Name:  "value",
				Value: math.NaN(),
			},
			want: "value NaN",
		},
		{
			name: "empty metric name",
			result: MetricResult{
				Name:  "",
				Value: 10,
			},
			want: " 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMetric(tt.result)

			if got != tt.want {
				t.Fatalf("formatMetric() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMetricsHandler(t *testing.T) {
	t.Run("empty store", func(t *testing.T) {
		store := NewMetricsStore()
		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if got := rec.Header().Get("Content-Type"); got != "text/plain; version=0.0.4" {
			t.Fatalf("Content-Type = %q, want %q",
				got,
				"text/plain; version=0.0.4",
			)
		}

		if got := rec.Code; got != http.StatusOK {
			t.Fatalf("status code = %d, want %d", got, http.StatusOK)
		}

		if got := rec.Body.String(); got != "" {
			t.Fatalf("body = %q, want empty body", got)
		}
	})

	t.Run("single metric without labels", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "requests_total",
			Description: "Total number of requests",
			Type:        "counter",
			Value:       42,
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP requests_total Total number of requests\n" +
			"# TYPE requests_total counter\n" +
			"requests_total 42\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("metrics are sorted by name", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "z_metric",
			Description: "Z metric",
			Type:        "gauge",
			Value:       3,
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "a_metric",
			Description: "A metric",
			Type:        "counter",
			Value:       1,
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "m_metric",
			Description: "M metric",
			Type:        "gauge",
			Value:       2,
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP a_metric A metric\n" +
			"# TYPE a_metric counter\n" +
			"a_metric 1\n" +
			"\n" +
			"# HELP m_metric M metric\n" +
			"# TYPE m_metric gauge\n" +
			"m_metric 2\n" +
			"\n" +
			"# HELP z_metric Z metric\n" +
			"# TYPE z_metric gauge\n" +
			"z_metric 3\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("samples with labels are sorted by label key", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        "counter",
			Value:       3,
			Labels: map[string]string{
				"method": "POST",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        "counter",
			Value:       1,
			Labels: map[string]string{
				"method": "GET",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        "counter",
			Value:       2,
			Labels: map[string]string{
				"method": "DELETE",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP http_requests_total Total HTTP requests\n" +
			"# TYPE http_requests_total counter\n" +
			`http_requests_total{method="DELETE"} 2` + "\n" +
			`http_requests_total{method="GET"} 1` + "\n" +
			`http_requests_total{method="POST"} 3` + "\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("samples with multiple labels are sorted deterministically", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        "counter",
			Value:       2,
			Labels: map[string]string{
				"status": "200",
				"method": "POST",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "http_requests_total",
			Description: "Total HTTP requests",
			Type:        "counter",
			Value:       1,
			Labels: map[string]string{
				"status": "200",
				"method": "GET",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP http_requests_total Total HTTP requests\n" +
			"# TYPE http_requests_total counter\n" +
			`http_requests_total{method="GET",status="200"} 1` + "\n" +
			`http_requests_total{method="POST",status="200"} 2` + "\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("sample without labels is sorted before labeled samples", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "metric",
			Description: "Test metric",
			Type:        "gauge",
			Value:       2,
			Labels: map[string]string{
				"source": "api",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "metric",
			Description: "Test metric",
			Type:        "gauge",
			Value:       1,
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP metric Test metric\n" +
			"# TYPE metric gauge\n" +
			"metric 1\n" +
			`metric{source="api"} 2` + "\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("special numeric values are formatted", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "positive_inf",
			Description: "Positive infinity",
			Type:        "gauge",
			Value:       math.Inf(1),
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "negative_inf",
			Description: "Negative infinity",
			Type:        "gauge",
			Value:       math.Inf(-1),
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "not_a_number",
			Description: "NaN",
			Type:        "gauge",
			Value:       math.NaN(),
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		want := "" +
			"# HELP negative_inf Negative infinity\n" +
			"# TYPE negative_inf gauge\n" +
			"negative_inf -Inf\n" +
			"\n" +
			"# HELP not_a_number NaN\n" +
			"# TYPE not_a_number gauge\n" +
			"not_a_number NaN\n" +
			"\n" +
			"# HELP positive_inf Positive infinity\n" +
			"# TYPE positive_inf gauge\n" +
			"positive_inf +Inf\n" +
			"\n"

		if got := rec.Body.String(); got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("store is not modified by handler", func(t *testing.T) {
		store := NewMetricsStore()

		if err := store.Set(MetricResult{
			Name:        "metric",
			Description: "Test",
			Type:        "gauge",
			Value:       1,
			Labels: map[string]string{
				"method": "GET",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		if err := store.Set(MetricResult{
			Name:        "metric",
			Description: "Test",
			Type:        "gauge",
			Value:       2,
			Labels: map[string]string{
				"method": "POST",
			},
		}); err != nil {
			t.Fatalf("failed to store metric: %v", err)
		}

		before := store.GetAll()

		handler := NewMetricsHandler(store)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		after := store.GetAll()

		if len(before) != len(after) {
			t.Fatalf(
				"store size changed: before=%d after=%d",
				len(before),
				len(after),
			)
		}

		for key, beforeMetric := range before {
			afterMetric, ok := after[key]
			if !ok {
				t.Fatalf("metric %q disappeared from store", key)
			}

			if beforeMetric.Name != afterMetric.Name ||
				beforeMetric.Description != afterMetric.Description ||
				beforeMetric.Type != afterMetric.Type ||
				beforeMetric.Value != afterMetric.Value {
				t.Fatalf("metric %q changed after handler execution", key)
			}

			if len(beforeMetric.Labels) != len(afterMetric.Labels) {
				t.Fatalf("labels for metric %q changed", key)
			}

			for labelKey, beforeValue := range beforeMetric.Labels {
				if afterMetric.Labels[labelKey] != beforeValue {
					t.Fatalf(
						"label %q of metric %q changed",
						labelKey,
						key,
					)
				}
			}
		}
	})
}

func TestNewMetricsHandler_AllowsGET(t *testing.T) {
	store := &MetricsStore{}

	handler := NewMetricsHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf(
			"GET /metrics returned status %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}

func TestNewMetricsHandler_RejectsNonGETMethods(t *testing.T) {
	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			store := &MetricsStore{}

			handler := NewMetricsHandler(store)

			req := httptest.NewRequest(method, "/metrics", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf(
					"%s /metrics returned status %d, want %d",
					method,
					rec.Code,
					http.StatusMethodNotAllowed,
				)
			}
		})
	}
}

func BenchmarkMetricsHandler(b *testing.B) {
	store := NewMetricsStore()

	metrics := []MetricResult{
		{
			Name:        "cpu_usage_percent",
			Description: "CPU usage percentage",
			Type:        "gauge",
			Value:       42.5,
		},
		{
			Name:        "memory_used_bytes",
			Description: "Used memory",
			Type:        "gauge",
			Value:       217456640,
		},
		{
			Name:        "network_receive_bytes_total",
			Description: "Network received bytes",
			Type:        "counter",
			Value:       9165713,
			Labels: map[string]string{
				"interface": "wlan0",
			},
		},
		{
			Name:        "network_transmit_bytes_total",
			Description: "Network transmitted bytes",
			Type:        "counter",
			Value:       472756,
			Labels: map[string]string{
				"interface": "wlan0",
			},
		},
		{
			Name:        "system_info",
			Description: "Static information about the host",
			Type:        "gauge",
			Value:       1,
			Labels: map[string]string{
				"hostname": "rpi3b",
				"os":       "linux",
				"arch":     "arm",
				"cpus":     "4",
			},
		},
		{
			Name:        "temperature_celsius",
			Description: "CPU temperature",
			Type:        "gauge",
			Value:       54.768,
		},
	}

	for _, metric := range metrics {
		if err := store.Set(metric); err != nil {
			b.Fatal(err)
		}
	}

	handler := NewMetricsHandler(store)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	}
}
