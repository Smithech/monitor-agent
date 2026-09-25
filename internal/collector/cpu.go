package collector

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"monitor-agent/internal/metrics"
)

type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

type CPUDeltas struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

type CPUStatsReader func() (CPUStats, error)

//! posible mejora
/*
Pero hay una pequeña consecuencia

En el caso de error de lectura estamos asignando:

previous = CPUStats{}

aunque realmente no tenemos una medición válida.

No es grave para el funcionamiento actual, porque en la siguiente iteración collectCPU utilizará el reader y, si funciona, calculará contra cero. Pero conceptualmente podemos hacerlo mejor.

La solución más limpia sería que collectCPU indicara explícitamente si obtuvo un current válido, por ejemplo:

func collectCPU(
    previous CPUStats,
    reader CPUStatsReader,
) (CPUStats, MetricResult, bool, error)

donde bool significaría algo como:

current válido: true
current no disponible: false
*/

func CollectCPU(
	previous CPUStats,
	reader CPUStatsReader,
) (CPUStats, metrics.MetricResult, error) {
	current, err := reader()

	if err != nil {
		return CPUStats{}, metrics.MetricResult{}, err
	}

	usage, err := calculateCPUUsage(previous, current)

	if err != nil {
		return current, metrics.MetricResult{}, err
	}

	return current, cpuUsageToMetric(usage), nil
}

func cpuUsageToMetric(usage float64) metrics.MetricResult {
	return metrics.MetricResult{
		Name:        "cpu_usage_percent",
		Description: "CPU usage percentage",
		Type:        "gauge",
		Value:       usage,
	}
}

func GetCPUStats() (CPUStats, error) {
	file, err := os.Open("/proc/stat")

	if err != nil {
		return CPUStats{}, fmt.Errorf("error opening /proc/stat: %w", err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 9 || parts[0] != "cpu" {
			continue
		}

		var values [8]uint64

		for i := range values {
			values[i], err = strconv.ParseUint(parts[i+1], 10, 64)

			if err != nil {
				return CPUStats{}, fmt.Errorf(
					"error parsing CPU stat %q: %w",
					parts[i+1],
					err,
				)
			}
		}

		return CPUStats{
			User:    values[0],
			Nice:    values[1],
			System:  values[2],
			Idle:    values[3],
			IOWait:  values[4],
			IRQ:     values[5],
			SoftIRQ: values[6],
			Steal:   values[7],
		}, nil
	}

	if err := scanner.Err(); err != nil {
		return CPUStats{}, fmt.Errorf("error reading /proc/stat: %w", err)
	}

	return CPUStats{}, fmt.Errorf("cpu stats not found")
}

func calculateCPUUsage(previous CPUStats, current CPUStats) (float64, error) {

	deltas, err := calculateCPUDeltas(previous, current)

	if err != nil {
		return 0, err
	}

	busyDelta := deltas.User + deltas.Nice + deltas.System + deltas.IRQ + deltas.SoftIRQ + deltas.Steal

	idleTime := deltas.Idle + deltas.IOWait

	totalDelta := busyDelta + idleTime

	if totalDelta == 0 {
		return 0, errors.New("total delta value is 0")
	}

	return float64(busyDelta) / float64(totalDelta) * 100, nil
}

func calculateCPUDeltas(previous CPUStats, current CPUStats) (CPUDeltas, error) {
	userDelta, err := calculateDelta(previous.User, current.User)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("userDelta - %w", err)
	}

	niceDelta, err := calculateDelta(previous.Nice, current.Nice)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("niceDelta - %w", err)
	}

	systemDelta, err := calculateDelta(previous.System, current.System)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("systemDelta - %w", err)
	}

	idleDelta, err := calculateDelta(previous.Idle, current.Idle)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("idleDelta - %w", err)
	}

	iowaitDelta, err := calculateDelta(previous.IOWait, current.IOWait)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("iowaitDelta - %w", err)
	}

	irqDelta, err := calculateDelta(previous.IRQ, current.IRQ)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("irqDelta - %w", err)
	}

	softIRQDelta, err := calculateDelta(previous.SoftIRQ, current.SoftIRQ)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("softIRQDelta - %w", err)
	}

	stealDelta, err := calculateDelta(previous.Steal, current.Steal)

	if err != nil {
		return CPUDeltas{}, fmt.Errorf("stealDelta - %w", err)
	}

	return CPUDeltas{
		User:    userDelta,
		Nice:    niceDelta,
		System:  systemDelta,
		Idle:    idleDelta,
		IOWait:  iowaitDelta,
		IRQ:     irqDelta,
		SoftIRQ: softIRQDelta,
		Steal:   stealDelta,
	}, nil
}

var errInvalidDelta = errors.New("invalid CPU counter delta")

func calculateDelta(previous uint64, current uint64) (uint64, error) {
	if current < previous {
		return 0, fmt.Errorf(
			"%w: current value (%d) is less than previous value (%d)",
			errInvalidDelta,
			current,
			previous,
		)
	}
	return current - previous, nil
}
