package collector

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"monitor-agent/internal/metrics"
)

var errSensorNotSupported = errors.New("temperature sensor not supported")
var errSensorUnavailable = errors.New("temperature sensor unavailable")

type TemperatureInfo struct {
	Celsius float64
}

type TemperatureInfoReader func() (TemperatureInfo, error)

func CollectTemperatureMetrics() ([]metrics.MetricResult, error) {
	return collectTemperature(readTemperature)
}

func collectTemperature(reader TemperatureInfoReader) ([]metrics.MetricResult, error) {
	temperatureInfo, err := reader()

	if err != nil {
		return nil, err
	}

	return []metrics.MetricResult{
		{
			Name:        "temperature_celsius",
			Description: "CPU temperature",
			Type:        "gauge",
			Value:       temperatureInfo.Celsius,
		},
	}, nil
}

var ErrTemperatureNotSupported = errors.New("temperature sensor not supported")

func readTemperature() (TemperatureInfo, error) {
	file, err := os.Open("/sys/class/thermal/thermal_zone0/temp")

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TemperatureInfo{}, ErrTemperatureNotSupported
		}
		return TemperatureInfo{}, errSensorUnavailable
	}

	defer file.Close()

	return getTemperature(file)
}

func getTemperature(reader io.Reader) (TemperatureInfo, error) {
	data, err := io.ReadAll(reader)

	if err != nil {
		return TemperatureInfo{}, fmt.Errorf("reading temperature: %w", errSensorUnavailable)
	}

	value := strings.TrimSpace(string(data))

	millidegrees, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		return TemperatureInfo{}, fmt.Errorf("invalid temperature %q: %w", value, errSensorUnavailable)
	}

	return TemperatureInfo{
		Celsius: float64(millidegrees) / 1000,
	}, nil
}

func temperatureInfoToMetric(info TemperatureInfo) metrics.MetricResult {
	return metrics.MetricResult{
		Name:        "temperature_celsius",
		Description: "CPU temperature",
		Type:        "gauge",
		Value:       info.Celsius,
	}
}
