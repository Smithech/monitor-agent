package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type requestStats struct {
	mu sync.Mutex

	requests     int
	successes    int
	failures     int
	totalLatency time.Duration
	maxLatency   time.Duration
	totalBytes   int64
}

type requestStatsSnapshot struct {
	requests     int
	successes    int
	failures     int
	totalLatency time.Duration
	maxLatency   time.Duration
	totalBytes   int64
}

func (s *requestStats) record(success bool, latency time.Duration, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests++
	if success {
		s.successes++
	} else {
		s.failures++
	}

	s.totalLatency += latency

	if latency > s.maxLatency {
		s.maxLatency = latency
	}

	s.totalBytes += bytes
}

func (s *requestStats) snapshot() requestStatsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return requestStatsSnapshot{
		requests:     s.requests,
		successes:    s.successes,
		failures:     s.failures,
		totalLatency: s.totalLatency,
		maxLatency:   s.maxLatency,
		totalBytes:   s.totalBytes,
	}
}

type memorySample struct {
	Time            time.Time
	RSSKB           int64
	MemoryCurrentKB int64
	CPUUsageNSec    int64
}

func main() {
	baseURL := flag.String("url", "http://127.0.0.1:8080", "monitor-agent base URL")
	duration := flag.Duration("duration", 30*time.Minute, "test duration")
	metricsInterval := flag.Duration("metrics-interval", 15*time.Second, "metrics request interval")
	healthInterval := flag.Duration("health-interval", 30*time.Second, "health request interval")
	readyInterval := flag.Duration("ready-interval", 30*time.Second, "ready request interval")
	sampleInterval := flag.Duration("sample-interval", 60*time.Second, "process resource sampling interval")
	output := flag.String("output", "loadtest-resources.csv", "CSV output file")

	flag.Parse()

	fmt.Println("monitor-agent load test")
	fmt.Println("-----------------------")
	fmt.Printf("URL:                %s\n", *baseURL)
	fmt.Printf("Duration:           %s\n", *duration)
	fmt.Printf("Metrics interval:   %s\n", *metricsInterval)
	fmt.Printf("Health interval:    %s\n", *healthInterval)
	fmt.Printf("Ready interval:     %s\n", *readyInterval)
	fmt.Printf("Resource interval:  %s\n", *sampleInterval)
	fmt.Printf("Output:             %s\n", *output)
	fmt.Println()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	metricsStats := &requestStats{}
	healthStats := &requestStats{}
	readyStats := &requestStats{}

	resourceSamples := make([]memorySample, 0)

	csvFile, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)

	if err := writer.Write([]string{
		"timestamp",
		"rss_kb",
		"memory_current_kb",
		"cpu_usage_nsec",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "write CSV header: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(4)

	go runEndpointLoop(
		ctx,
		&wg,
		client,
		*baseURL+"/metrics",
		*metricsInterval,
		metricsStats,
	)

	go runEndpointLoop(
		ctx,
		&wg,
		client,
		*baseURL+"/health",
		*healthInterval,
		healthStats,
	)

	go runEndpointLoop(
		ctx,
		&wg,
		client,
		*baseURL+"/ready",
		*readyInterval,
		readyStats,
	)

	go runResourceSampler(
		ctx,
		&wg,
		writer,
		&resourceSamples,
		*sampleInterval,
	)

	start := time.Now()

	fmt.Println("Test running...")
	fmt.Println("Press Ctrl+C to stop early.")
	fmt.Println()

	<-ctx.Done()

	wg.Wait()

	durationActual := time.Since(start)

	writer.Flush()

	if err := writer.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "CSV error: %v\n", err)
	}

	printSummary(
		durationActual,
		"metrics",
		metricsStats.snapshot(),
	)

	printSummary(
		durationActual,
		"health",
		healthStats.snapshot(),
	)

	printSummary(
		durationActual,
		"ready",
		readyStats.snapshot(),
	)

	printResourceSummary(resourceSamples)

	fmt.Printf("\nDetailed resource samples: %s\n", *output)
}

func runEndpointLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	client *http.Client,
	url string,
	interval time.Duration,
	stats *requestStats,
) {
	defer wg.Done()

	request := func() {
		start := time.Now()

		resp, err := client.Get(url)
		if err != nil {
			stats.record(false, time.Since(start), 0)
			return
		}

		bytes, _ := readAndDiscard(resp)
		latency := time.Since(start)

		success := resp.StatusCode >= 200 && resp.StatusCode < 300

		stats.record(success, latency, bytes)

		resp.Body.Close()
	}

	request()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			request()
		}
	}
}

func readAndDiscard(resp *http.Response) (int64, error) {
	return io.Copy(io.Discard, resp.Body)
}

func runResourceSampler(
	ctx context.Context,
	wg *sync.WaitGroup,
	writer *csv.Writer,
	samples *[]memorySample,
	interval time.Duration,
) {
	defer wg.Done()

	sample := func() {
		rssKB, memoryCurrentKB, cpuUsageNSec, err := readProcessStats()
		if err != nil {
			fmt.Printf("[%s] resource sample error: %v\n",
				time.Now().Format(time.RFC3339),
				err,
			)
			return
		}

		now := time.Now()

		*samples = append(*samples, memorySample{
			Time:            now,
			RSSKB:           rssKB,
			MemoryCurrentKB: memoryCurrentKB,
			CPUUsageNSec:    cpuUsageNSec,
		})

		_ = writer.Write([]string{
			now.Format(time.RFC3339),
			strconv.FormatInt(rssKB, 10),
			strconv.FormatInt(memoryCurrentKB, 10),
			strconv.FormatInt(cpuUsageNSec, 10),
		})

		writer.Flush()

		fmt.Printf(
			"[%s] RSS=%d KB MemoryCurrent=%d KB CPU=%d ns\n",
			now.Format("15:04:05"),
			rssKB,
			memoryCurrentKB,
			cpuUsageNSec,
		)
	}

	sample()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			sample()
		}
	}
}

func readProcessStats() (int64, int64, int64, error) {
	output, err := exec.Command(
		"systemctl",
		"show",
		"monitor-agent",
		"-p",
		"MainPID",
		"-p",
		"MemoryCurrent",
		"-p",
		"CPUUsageNSec",
	).Output()

	if err != nil {
		return 0, 0, 0, err
	}

	values := make(map[string]int64)

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		value, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		values[parts[0]] = value
	}

	pid := values["MainPID"]
	if pid == 0 {
		return 0, 0, 0, fmt.Errorf("monitor-agent MainPID not found")
	}

	psOutput, err := exec.Command(
		"ps",
		"-o",
		"rss=",
		"-p",
		strconv.FormatInt(pid, 10),
	).Output()

	if err != nil {
		return 0, 0, 0, err
	}

	rssKB, err := strconv.ParseInt(
		strings.TrimSpace(string(psOutput)),
		10,
		64,
	)
	if err != nil {
		return 0, 0, 0, err
	}

	memoryCurrentKB := values["MemoryCurrent"] / 1024

	return rssKB, memoryCurrentKB, values["CPUUsageNSec"], nil
}

func printSummary(
	duration time.Duration,
	name string,
	stats requestStatsSnapshot,
) {
	fmt.Printf("\n[%s]\n", name)
	fmt.Printf("requests:       %d\n", stats.requests)
	fmt.Printf("successes:      %d\n", stats.successes)
	fmt.Printf("failures:       %d\n", stats.failures)
	fmt.Printf("bytes:          %d\n", stats.totalBytes)

	if stats.requests > 0 {
		average := stats.totalLatency / time.Duration(stats.requests)

		fmt.Printf("avg latency:    %s\n", average)
		fmt.Printf("max latency:    %s\n", stats.maxLatency)
		fmt.Printf(
			"requests/min:   %.2f\n",
			float64(stats.requests)/duration.Minutes(),
		)
	}
}

func printResourceSummary(samples []memorySample) {
	if len(samples) == 0 {
		fmt.Println("\nNo resource samples collected.")
		return
	}

	first := samples[0]
	last := samples[len(samples)-1]

	fmt.Println("\n[process resources]")
	fmt.Printf("samples:            %d\n", len(samples))
	fmt.Printf("initial RSS:        %d KB\n", first.RSSKB)
	fmt.Printf("final RSS:          %d KB\n", last.RSSKB)
	fmt.Printf("RSS change:         %+d KB\n", last.RSSKB-first.RSSKB)

	fmt.Printf("initial cgroup mem: %d KB\n", first.MemoryCurrentKB)
	fmt.Printf("final cgroup mem:   %d KB\n", last.MemoryCurrentKB)
	fmt.Printf("memory change:      %+d KB\n",
		last.MemoryCurrentKB-first.MemoryCurrentKB,
	)

	fmt.Printf("CPU accumulated:    %.3f s\n",
		float64(last.CPUUsageNSec)/1e9,
	)

	if len(samples) >= 2 {
		elapsed := last.Time.Sub(first.Time)
		cpuDelta := last.CPUUsageNSec - first.CPUUsageNSec

		fmt.Printf(
			"CPU during test:    %.3f s\n",
			float64(cpuDelta)/1e9,
		)

		if elapsed > 0 {
			cpuPercent := float64(cpuDelta) /
				float64(elapsed.Nanoseconds()) *
				100

			fmt.Printf(
				"CPU average:        %.3f%%\n",
				cpuPercent,
			)
		}
	}
}
