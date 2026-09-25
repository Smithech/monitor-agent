package metrics

import "testing"

func TestValidateMetric(t *testing.T) {
	tests := []struct {
		name    string
		metric  MetricResult
		wantErr bool
	}{
		{
			name: "valid gauge",
			metric: MetricResult{
				Name:  "cpu_usage_percent",
				Type:  "gauge",
				Value: 42,
			},
		},
		{
			name: "valid counter",
			metric: MetricResult{
				Name:  "requests_total",
				Type:  "counter",
				Value: 100,
			},
		},
		{
			name: "empty name",
			metric: MetricResult{
				Name: "",
				Type: "gauge",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			metric: MetricResult{
				Name: "temperature",
				Type: "something",
			},
			wantErr: true,
		},
		{
			name: "empty description is allowed",
			metric: MetricResult{
				Name: "temperature",
				Type: "gauge",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetric(tt.metric)

			if (err != nil) != tt.wantErr {
				t.Fatalf("validateMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidLabelName(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  bool
	}{
		{
			name:  "valid name",
			label: "interface",
			want:  true,
		},
		{
			name:  "valid name with underscore",
			label: "status_code",
			want:  true,
		},
		{
			name:  "valid name starting with underscore",
			label: "_foo",
			want:  true,
		},
		{
			name:  "valid name with numbers",
			label: "status200",
			want:  true,
		},
		{
			name:  "invalid empty name",
			label: "",
			want:  false,
		},
		{
			name:  "invalid name starting with numbers",
			label: "123status",
			want:  false,
		},
		{
			name:  "invalid name with hyphen",
			label: "http-status",
			want:  false,
		},
		{
			name:  "invalid name with space",
			label: "http status",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidLabelName(tt.label)

			if got != tt.want {
				t.Errorf("IsValidLabelName(%q) = %t, want %t", tt.label, got, tt.want)
			}
		})
	}
}

func TestValidateMetric_InvalidType(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		wantErr    string
	}{
		{
			name:       "invalid type",
			metricType: "invalid",
			wantErr:    `invalid metric type "invalid"`,
		},
		{
			name:       "empty type",
			metricType: "",
			wantErr:    `invalid metric type ""`,
		},
		{
			name:       "unsupported type",
			metricType: "histogram",
			wantErr:    `invalid metric type "histogram"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := MetricResult{
				Name:   "test_metric",
				Type:   tt.metricType,
				Labels: map[string]string{},
			}

			err := validateMetric(metric)

			if err == nil {
				t.Fatalf("validateMetric() expected an error, got nil")
			}

			if err.Error() != tt.wantErr {
				t.Errorf("validateMetric() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
