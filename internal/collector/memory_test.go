package collector

import (
	"errors"
	"strings"
	"testing"
)

func TestCollectMemory(t *testing.T) {
	fakeReader := func() (MemoryInfo, error) {
		return MemoryInfo{
			TotalBytes:     1000,
			AvailableBytes: 300,
			UsedBytes:      700,
		}, nil
	}

	got, err := collectMemory(fakeReader)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if count := len(got); count != 3 {
		t.Errorf("expected 3 metrics, got %d", count)
	}

	expected := []struct {
		name        string
		description string
		metricType  string
		value       float64
	}{

		{
			name:        "memory_total_bytes",
			description: "Total memory",
			metricType:  "gauge",
			value:       1000,
		},
		{
			name:        "memory_available_bytes",
			description: "Available memory",
			metricType:  "gauge",
			value:       300,
		},
		{
			name:        "memory_used_bytes",
			description: "Used memory",
			metricType:  "gauge",
			value:       700,
		},
	}

	for i, tt := range expected {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric := got[i]

			if gotMetric.Name != tt.name {
				t.Errorf("expected name %q, got %q", tt.name, gotMetric.Name)
			}

			if gotMetric.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, gotMetric.Description)
			}

			if gotMetric.Type != tt.metricType {
				t.Errorf("expected type %q, got %q", tt.metricType, gotMetric.Type)
			}

			if gotMetric.Value != tt.value {
				t.Errorf("expected value %f, got %f", tt.value, gotMetric.Value)
			}
		})
	}
}

func TestCollectMemoryReaderError(t *testing.T) {
	expectedErr := errors.New("reader failed")

	fakeReader := func() (MemoryInfo, error) {
		return MemoryInfo{}, expectedErr
	}

	got, err := collectMemory(fakeReader)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected err %v, got %v", expectedErr, err)
	}

	if count := len(got); count != 0 {
		t.Errorf("expected 0 metrics, got %d", count)
	}
}

func TestGetMemoryInfo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    MemoryInfo
		wantErr string
	}{
		{
			name: "success",
			content: `MemTotal:        1000 kB
					  MemFree:          700 kB
					  MemAvailable:     300 kB
					  `,
			want: MemoryInfo{
				TotalBytes:     1000 * 1024,
				AvailableBytes: 300 * 1024,
				UsedBytes:      700 * 1024,
			},
		},
		{
			name: "ignores unrelated metrics",
			content: `MemFree:           700 kB
                      Buffers:           100 kB
                      Cached:            200 kB
                      MemTotal:         1000 kB
                      MemAvailable:      300 kB
                      SwapTotal:        2000 kB
                      `,
			want: MemoryInfo{
				TotalBytes:     1000 * 1024,
				AvailableBytes: 300 * 1024,
				UsedBytes:      700 * 1024,
			},
		},
		{
			name: "missing MemTotal",
			content: `MemFree:          700 kB
                      MemAvailable:     300 kB
                      `,
			wantErr: "metric not found: MemTotal",
		},
		{
			name: "missing MemAvailable",
			content: `MemTotal:        1000 kB
                      MemFree:          700 kB
                      `,
			wantErr: "metric not found: MemAvailable",
		},
		{
			name: "invalid MemTotal",
			content: `MemTotal:        invalid kB
                      MemAvailable:     300 kB
                      `,
			wantErr: "parsing MemTotal:",
		},
		{
			name: "invalid MemAvailable",
			content: `MemTotal:        1000 kB
                      MemAvailable:     invalid kB
                      `,
			wantErr: "parsing MemAvailable:",
		},
		{
			name: "ignores malformed lines",
			content: `invalid
                      MemTotal:
                      MemTotal: 1000 kB
                      MemAvailable: 300 kB
                      `,
			want: MemoryInfo{
				TotalBytes:     1000 * 1024,
				AvailableBytes: 300 * 1024,
				UsedBytes:      700 * 1024,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)

			got, err := getMemoryInfo(reader)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

func TestGetMemoryInfo_ReaderError(t *testing.T) {
	_, err := getMemoryInfo(errorReader{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "error reading /proc/meminfo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func BenchmarkGetMemoryInfo(b *testing.B) {
	data := `MemTotal:        947048 kB
MemFree:          100000 kB
MemAvailable:     729591 kB
Buffers:           50000 kB
Cached:           100000 kB
SwapCached:            0 kB
Active:           300000 kB
Inactive:         200000 kB
`

	for i := 0; i < b.N; i++ {
		_, err := getMemoryInfo(strings.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}
