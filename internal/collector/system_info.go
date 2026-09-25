package collector

import (
	"fmt"
	"os"
	"runtime"

	"monitor-agent/internal/metrics"
)

type SystemInfo struct {
	Hostname string
	OS       string
	Arch     string
	CPUs     int
}

type SystemInfoReader func() (SystemInfo, error)

func collectSystemInfoFromOS() (SystemInfo, error) {
	hostname, err := os.Hostname()

	if err != nil {
		return SystemInfo{}, err
	}

	return SystemInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
	}, nil
}

func collectSystemInfo(reader SystemInfoReader) ([]metrics.MetricResult, error) {
	info, err := reader()

	if err != nil {
		return nil, err
	}

	return []metrics.MetricResult{
		systemInfoToMetric(info),
	}, nil
}

func systemInfoToMetric(info SystemInfo) metrics.MetricResult {
	return metrics.MetricResult{
		Name:        "system_info",
		Description: "Static information about the host",
		Type:        "gauge",
		Value:       1,
		Labels: map[string]string{
			"hostname": info.Hostname,
			"os":       info.OS,
			"arch":     info.Arch,
			"cpus":     fmt.Sprintf("%d", info.CPUs),
		},
	}
}

func CollectSystemInfoMetrics() ([]metrics.MetricResult, error) {
	return collectSystemInfo(collectSystemInfoFromOS)
}
