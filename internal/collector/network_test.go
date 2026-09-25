package collector

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"monitor-agent/internal/metrics"
)

func TestCollectNetwork(t *testing.T) {
	tests := []struct {
		name    string
		reader  NetworkStatsReader
		want    []metrics.MetricResult
		wantErr string
	}{
		{
			name: "success",
			reader: func() ([]networkStats, error) {
				return []networkStats{
					{
						Interface:     "eth0",
						ReceiveBytes:  123456,
						TransmitBytes: 654321,
					},
				}, nil
			},
			want: []metrics.MetricResult{
				{
					Name:        "network_receive_bytes_total",
					Description: "Network received bytes",
					Type:        "counter",
					Value:       123456,
					Labels: map[string]string{
						"interface": "eth0",
					},
				},
				{
					Name:        "network_transmit_bytes_total",
					Description: "Network transmitted bytes",
					Type:        "counter",
					Value:       654321,
					Labels: map[string]string{
						"interface": "eth0",
					},
				},
			},
		},
		{
			name: "multiple interfaces",
			reader: func() ([]networkStats, error) {
				return []networkStats{
					{
						Interface:     "eth0",
						ReceiveBytes:  1000,
						TransmitBytes: 2000,
					},
					{
						Interface:     "lo",
						ReceiveBytes:  3000,
						TransmitBytes: 4000,
					},
				}, nil
			},
			want: []metrics.MetricResult{
				{
					Name:        "network_receive_bytes_total",
					Description: "Network received bytes",
					Type:        "counter",
					Value:       1000,
					Labels: map[string]string{
						"interface": "eth0",
					},
				},
				{
					Name:        "network_transmit_bytes_total",
					Description: "Network transmitted bytes",
					Type:        "counter",
					Value:       2000,
					Labels: map[string]string{
						"interface": "eth0",
					},
				},
				{
					Name:        "network_receive_bytes_total",
					Description: "Network received bytes",
					Type:        "counter",
					Value:       3000,
					Labels: map[string]string{
						"interface": "lo",
					},
				},
				{
					Name:        "network_transmit_bytes_total",
					Description: "Network transmitted bytes",
					Type:        "counter",
					Value:       4000,
					Labels: map[string]string{
						"interface": "lo",
					},
				},
			},
		},
		{
			name: "empty stats",
			reader: func() ([]networkStats, error) {
				return []networkStats{}, nil
			},
			want: []metrics.MetricResult{},
		},
		{
			name: "reader error",
			reader: func() ([]networkStats, error) {
				return nil, errors.New("network reader error")
			},
			wantErr: "network reader error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectNetwork(tt.reader)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf(
						"expected error containing %q, got %q",
						tt.wantErr,
						err.Error(),
					)
				}

				if got != nil {
					t.Fatalf("expected nil metrics, got %+v", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestGetNetworkStats(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []networkStats
		wantErr string
	}{
		{
			name: "success",
			content: `Inter-|   Receive                                                |  Transmit
                      face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
                      eth0: 123456 100 0 0 0 0 0 0 654321 200 0 0 0 0 0 0
 					  lo:   1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0
                    `,
			want: []networkStats{
				{
					Interface:     "eth0",
					ReceiveBytes:  123456,
					TransmitBytes: 654321,
				},
				{
					Interface:     "lo",
					ReceiveBytes:  1000,
					TransmitBytes: 2000,
				},
			},
		},
		{
			name: "ignores invalid lines",
			content: `Inter-|   Receive                                                |  Transmit
    				  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
   					  invalid line
   					  another invalid line
   					  eth0: 123456 100 0 0 0 0 0 0 654321 200 0 0 0 0 0 0
   					`,
			want: []networkStats{
				{
					Interface:     "eth0",
					ReceiveBytes:  123456,
					TransmitBytes: 654321,
				},
			},
		},
		{
			name: "ignores lines with less than nine fields",
			content: `eth0: 123456 100 0 0
                      lo: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0
                    `,
			want: []networkStats{
				{
					Interface:     "lo",
					ReceiveBytes:  1000,
					TransmitBytes: 2000,
				},
			},
		},
		{
			name: "ignores lines without interface separator",
			content: `eth0 123456 100 0 0 0 0 0 0 654321 200 0 0 0 0 0 0
					  lo: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0
					`,
			want: []networkStats{
				{
					Interface:     "lo",
					ReceiveBytes:  1000,
					TransmitBytes: 2000,
				},
			},
		},
		{
			name:    "invalid receive bytes",
			content: `eth0: invalid 100 0 0 0 0 0 0 654321 200 0 0 0 0 0 0`,
			wantErr: "parsing receive bytes for eth0:",
		},
		{
			name:    "invalid transmit bytes",
			content: `eth0: 123456 100 0 0 0 0 0 0 invalid 200 0 0 0 0 0 0`,
			wantErr: "parsing transmit bytes for eth0:",
		},
		{
			name:    "empty input",
			content: "",
			want:    []networkStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)

			got, err := getNetworkStats(reader)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf(
						"expected error containing %q, got %q",
						tt.wantErr,
						err.Error(),
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestNetworkStatsToMetrics(t *testing.T) {
	stats := networkStats{
		Interface:     "eth0",
		ReceiveBytes:  123456,
		TransmitBytes: 654321,
	}

	got := networkStatsToMetrics(stats)

	if len(got) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(got))
	}

	tests := []struct {
		name      string
		metric    metrics.MetricResult
		wantName  string
		wantDesc  string
		wantType  string
		wantValue float64
		wantIface string
	}{
		{
			name:      "receive",
			metric:    got[0],
			wantName:  "network_receive_bytes_total",
			wantDesc:  "Network received bytes",
			wantType:  "counter",
			wantValue: 123456,
			wantIface: "eth0",
		},
		{
			name:      "transmit",
			metric:    got[1],
			wantName:  "network_transmit_bytes_total",
			wantDesc:  "Network transmitted bytes",
			wantType:  "counter",
			wantValue: 654321,
			wantIface: "eth0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metric.Name != tt.wantName {
				t.Errorf(
					"expected name %q, got %q",
					tt.wantName,
					tt.metric.Name,
				)
			}

			if tt.metric.Description != tt.wantDesc {
				t.Errorf(
					"expected description %q, got %q",
					tt.wantDesc,
					tt.metric.Description,
				)
			}

			if tt.metric.Type != tt.wantType {
				t.Errorf(
					"expected type %q, got %q",
					tt.wantType,
					tt.metric.Type,
				)
			}

			if tt.metric.Value != tt.wantValue {
				t.Errorf(
					"expected value %f, got %f",
					tt.wantValue,
					tt.metric.Value,
				)
			}

			if tt.metric.Labels["interface"] != tt.wantIface {
				t.Errorf(
					"expected interface label %q, got %q",
					tt.wantIface,
					tt.metric.Labels["interface"],
				)
			}
		})
	}
}

func TestGetNetworkStats_ReaderError(t *testing.T) {
	_, err := getNetworkStats(errorReader{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "error reading /proc/net/dev") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func BenchmarkGetNetworkStats(b *testing.B) {
	data := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo:    9494     100    0    0    0     0          0         0     9494     100    0    0    0     0       0          0
  wlan0: 9165713   12000    0    0    0     0          0         0   472756    9000    0    0    0     0       0          0
`

	for i := 0; i < b.N; i++ {
		_, err := getNetworkStats(strings.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}
