package collector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"monitor-agent/internal/metrics"
)

type networkStats struct {
	Interface     string
	ReceiveBytes  uint64
	TransmitBytes uint64
}

type NetworkStatsReader func() ([]networkStats, error)

func CollectNetworkMetrics() ([]metrics.MetricResult, error) {
	return collectNetwork(readNetworkInfo)
}

func collectNetwork(reader NetworkStatsReader) ([]metrics.MetricResult, error) {
	stats, err := reader()

	if err != nil {
		return nil, err
	}

	metrics := make([]metrics.MetricResult, 0)

	for _, stat := range stats {
		metrics = append(metrics, networkStatsToMetrics(stat)...)
	}

	return metrics, nil
}

func readNetworkInfo() ([]networkStats, error) {
	file, err := os.Open("/proc/net/dev")

	if err != nil {
		return []networkStats{}, fmt.Errorf("error opening /proc/net/dev: %w", err)
	}

	defer file.Close()

	return getNetworkStats(file)
}

func getNetworkStats(reader io.Reader) ([]networkStats, error) {
	scanner := bufio.NewScanner(reader)

	statsList := make([]networkStats, 0)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 10 || !strings.HasSuffix(parts[0], ":") {
			continue
		}

		interfaceName := strings.TrimSuffix(parts[0], ":")
		receiveBytes, err := strconv.ParseUint(parts[1], 10, 64)

		if err != nil {
			return nil, fmt.Errorf("parsing receive bytes for %s: %w", interfaceName, err)
		}

		transmitBytes, err := strconv.ParseUint(parts[9], 10, 64)

		if err != nil {
			return nil, fmt.Errorf("parsing transmit bytes for %s: %w", interfaceName, err)
		}

		statsList = append(statsList, networkStats{
			Interface:     interfaceName,
			ReceiveBytes:  receiveBytes,
			TransmitBytes: transmitBytes,
		})
	}

	if scanner.Err() != nil {
		return []networkStats{}, fmt.Errorf("error reading /proc/net/dev: %w", scanner.Err())
	}

	return statsList, nil
}

func networkStatsToMetrics(stats networkStats) []metrics.MetricResult {
	return []metrics.MetricResult{
		{
			Name:        "network_receive_bytes_total",
			Description: "Network received bytes",
			Type:        "counter",
			Value:       float64(stats.ReceiveBytes),
			Labels: map[string]string{
				"interface": stats.Interface,
			},
		},
		{
			Name:        "network_transmit_bytes_total",
			Description: "Network transmitted bytes",
			Type:        "counter",
			Value:       float64(stats.TransmitBytes),
			Labels: map[string]string{
				"interface": stats.Interface,
			},
		},
	}
}
