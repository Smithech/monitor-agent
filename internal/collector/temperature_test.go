package collector

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"monitor-agent/internal/metrics"
)

func TestGetTemperature(t *testing.T) {
	tests := []struct {
		name      string
		reader    io.Reader
		want      float64
		wantError bool
	}{
		{
			name:   "46160",
			reader: strings.NewReader("46160\n"),
			want:   46.16,
		},
		{
			name:   "42000",
			reader: strings.NewReader("42000\n"),
			want:   42.0,
		},
		{
			name:   "negative temperature",
			reader: strings.NewReader("-1000\n"),
			want:   -1.0,
		},
		{
			name:      "invalid value",
			reader:    strings.NewReader("invalid\n"),
			wantError: true,
		},
		// errorReader.Read declarated at memory_test.go
		{
			name:      "reader with error",
			reader:    errorReader{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTemperature(tt.reader)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Celsius != tt.want {
				t.Fatalf("expected %.2f°C, got %.2f°C", tt.want, got.Celsius)
			}
		})
	}
}

func TestTemperatureInfoToMetric(t *testing.T) {
	info := TemperatureInfo{
		Celsius: 46.16,
	}

	got := temperatureInfoToMetric(info)

	want := metrics.MetricResult{
		Name:        "temperature_celsius",
		Description: "CPU temperature",
		Type:        "gauge",
		Value:       46.16,
	}

	if got.Name != want.Name {
		t.Fatalf("expected name %s, got %s", want.Name, got.Name)
	}

	if got.Description != want.Description {
		t.Fatalf("expected description %s, got %s", want.Description, got.Description)
	}

	if got.Type != want.Type {
		t.Fatalf("expected metric type %s, got %s", want.Type, got.Type)
	}

	if got.Value != want.Value {
		t.Fatalf("expected value %f, got %f", want.Value, got.Value)
	}
}

func TestCollectTemperature(t *testing.T) {
	tests := []struct {
		name      string
		reader    TemperatureInfoReader
		want      metrics.MetricResult
		wantError bool
	}{
		{
			name: "temperatura válida",
			reader: func() (TemperatureInfo, error) {
				return TemperatureInfo{
					Celsius: 46.16,
				}, nil
			},
			want: metrics.MetricResult{
				Name:        "temperature_celsius",
				Description: "CPU temperature",
				Type:        "gauge",
				Value:       46.16,
			},
		},
		{
			name: "error del reader",
			reader: func() (TemperatureInfo, error) {
				return TemperatureInfo{}, errors.New("temperature reader error")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectTemperature(tt.reader)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != 1 {
				t.Fatalf("expected 1 metric, got %d", len(got))
			}

			if got[0].Name != tt.want.Name {
				t.Fatalf(
					"expected name %s, got %s",
					tt.want.Name,
					got[0].Name,
				)
			}

			if got[0].Description != tt.want.Description {
				t.Fatalf(
					"expected description %s, got %s",
					tt.want.Description,
					got[0].Description,
				)
			}

			if got[0].Type != tt.want.Type {
				t.Fatalf(
					"expected type %s, got %s",
					tt.want.Type,
					got[0].Type,
				)
			}

			if got[0].Value != tt.want.Value {
				t.Fatalf(
					"expected value %.2f, got %.2f",
					tt.want.Value,
					got[0].Value,
				)
			}
		})
	}
}

func TestSensorErrorsAreDistinguishable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "sensor not supported",
			err:      errSensorNotSupported,
			expected: fmt.Errorf("reading sensor: %w", errSensorNotSupported),
		},
		{
			name:     "sensor unavailable",
			err:      errSensorUnavailable,
			expected: fmt.Errorf("reading sensor: %w", errSensorUnavailable),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.expected, tt.err) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", tt.err, tt.expected)
			}
		})
	}
}

func TestSensorErrorsAreNotConfused(t *testing.T) {
	if errors.Is(errSensorNotSupported, errSensorUnavailable) {
		t.Fatal("errSensorNotSupported must not match errSensorUnavailable")
	}

	if errors.Is(errSensorUnavailable, errSensorNotSupported) {
		t.Fatal("errSensorUnavailable must not match errSensorNotSupported")
	}
}

func TestGetTemperature_ReadError(t *testing.T) {
	_, err := getTemperature(errorReader{})

	if err == nil {
		t.Fatal("expected an error")
	}

	if !errors.Is(err, errSensorUnavailable) {
		t.Errorf("expected errSensorUnavailable, got %v", err)
	}

	if !strings.Contains(err.Error(), "reading temperature") {
		t.Errorf("expected error to contain %q, got %q",
			"reading temperature", err.Error())
	}
}

func TestGetTemperature_InvalidTemperature(t *testing.T) {
	_, err := getTemperature(strings.NewReader("hello"))

	if err == nil {
		t.Fatal("expected an error")
	}

	if !errors.Is(err, errSensorUnavailable) {
		t.Errorf("expected errSensorUnavailable, got %v", err)
	}

	if !strings.Contains(err.Error(), `invalid temperature "hello"`) {
		t.Errorf("expected error to contain %q, got %q",
			`invalid temperature "hello"`, err.Error())
	}
}

func BenchmarkGetTemperature(b *testing.B) {
	data := "54768\n"

	for i := 0; i < b.N; i++ {
		_, err := getTemperature(strings.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}
