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

type MemoryInfo struct {
	TotalBytes     uint64
	AvailableBytes uint64
	UsedBytes      uint64
}

type MemoryStatsReader func() (MemoryInfo, error)

func CollectMemoryMetrics() ([]metrics.MetricResult, error) {
	return collectMemory(readMemoryInfo)
}

func collectMemory(reader MemoryStatsReader) ([]metrics.MetricResult, error) {
	memoryInfo, err := reader()

	if err != nil {
		return nil, err
	}

	return []metrics.MetricResult{
		{
			Name:        "memory_total_bytes",
			Description: "Total memory",
			Type:        "gauge",
			Value:       float64(memoryInfo.TotalBytes),
		},
		{
			Name:        "memory_available_bytes",
			Description: "Available memory",
			Type:        "gauge",
			Value:       float64(memoryInfo.AvailableBytes),
		},
		{
			Name:        "memory_used_bytes",
			Description: "Used memory",
			Type:        "gauge",
			Value:       float64(memoryInfo.UsedBytes),
		},
	}, nil
}

func readMemoryInfo() (MemoryInfo, error) {
	file, err := os.Open("/proc/meminfo")

	if err != nil {
		return MemoryInfo{}, fmt.Errorf("error opening /proc/meminfo: %w", err)
	}

	defer file.Close()

	return getMemoryInfo(file)
}

func getMemoryInfo(reader io.Reader) (MemoryInfo, error) {
	scanner := bufio.NewScanner(reader)

	var memTotal uint64
	var memAvailable uint64

	var foundMemTotal bool
	var foundMemAvailable bool

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 3 {
			continue
		}

		metricName := strings.TrimSuffix(parts[0], ":")

		if metricName != "MemTotal" && metricName != "MemAvailable" {
			continue
		}

		parsedValue, err := strconv.ParseUint(parts[1], 10, 64)

		if err != nil {
			return MemoryInfo{}, fmt.Errorf(
				"parsing %s: %w",
				metricName,
				err,
			)
		}

		switch metricName {
		case "MemTotal":
			memTotal = parsedValue
			foundMemTotal = true

		case "MemAvailable":
			memAvailable = parsedValue
			foundMemAvailable = true
		}
	}

	if err := scanner.Err(); err != nil {
		return MemoryInfo{}, fmt.Errorf("error reading /proc/meminfo: %w", err)
	}

	if !foundMemTotal {
		return MemoryInfo{}, fmt.Errorf("metric not found: MemTotal")
	}

	if !foundMemAvailable {
		return MemoryInfo{}, fmt.Errorf("metric not found: MemAvailable")
	}

	return MemoryInfo{
		TotalBytes:     memTotal * 1024,
		AvailableBytes: memAvailable * 1024,
		UsedBytes:      (memTotal - memAvailable) * 1024,
	}, nil
}
