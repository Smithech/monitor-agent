package collector

import (
	"errors"
	"testing"
)

func TestSystemInfoToMetric(t *testing.T) {
	info := SystemInfo{
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		CPUs:     8,
	}

	result := systemInfoToMetric(info)

	expectedLabels := map[string]string{
		"hostname": "test-host",
		"os":       "linux",
		"arch":     "amd64",
		"cpus":     "8",
	}

	if result.Name != "system_info" {
		t.Errorf("Name = %q, want %q", result.Name, "system_info")
	}

	if result.Description != "Static information about the host" {
		t.Errorf(
			"Description = %q, want %q",
			result.Description,
			"Static information about the host",
		)
	}

	if result.Type != "gauge" {
		t.Errorf("Type = %q, want %q", result.Type, "gauge")
	}

	if result.Value != 1 {
		t.Errorf("Value = %v, want %v", result.Value, 1)
	}

	for key, expected := range expectedLabels {
		if result.Labels[key] != expected {
			t.Errorf(
				"Labels[%q] = %q, want %q",
				key,
				result.Labels[key],
				expected,
			)
		}
	}
}

func TestCollectSystemInfoPropagatesReaderError(t *testing.T) {
	expectedErr := errors.New("reader error")

	reader := func() (SystemInfo, error) {
		return SystemInfo{}, expectedErr
	}

	result, err := collectSystemInfo(reader)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}
