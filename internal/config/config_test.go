package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()

	want := Config{
		CPUInterval:         3 * time.Second,
		MemoryInterval:      30 * time.Second,
		NetworkInterval:     30 * time.Second,
		TemperatureInterval: 30 * time.Second,
		RetryInterval:       1 * time.Second,
	}

	if got != want {
		t.Errorf("DefaultConfig() = %+v, want %+v", got, want)
	}
}

func TestConfigValidate_Valid(t *testing.T) {
	config := Config{
		CPUInterval:         3 * time.Second,
		MemoryInterval:      30 * time.Second,
		NetworkInterval:     30 * time.Second,
		TemperatureInterval: 30 * time.Second,
		RetryInterval:       1 * time.Second,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestConfigValidate_ZeroIntervals(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "CPUInterval",
			config: Config{
				CPUInterval:         0,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "MemoryInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      0,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "NetworkInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     0,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "TemperatureInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 0,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "RetryInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err == nil {
				t.Errorf("Validate() expected error for %s = 0", tt.name)
			}
		})
	}
}

func TestConfigValidate_NegativeIntervals(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "CPUInterval",
			config: Config{
				CPUInterval:         -1 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "MemoryInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      -1 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "NetworkInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     -1 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "TemperatureInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: -1 * time.Second,
				RetryInterval:       1 * time.Second,
			},
		},
		{
			name: "RetryInterval",
			config: Config{
				CPUInterval:         3 * time.Second,
				MemoryInterval:      30 * time.Second,
				NetworkInterval:     30 * time.Second,
				TemperatureInterval: 30 * time.Second,
				RetryInterval:       -1 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err == nil {
				t.Errorf("Validate() expected error for %s < 0", tt.name)
			}
		})
	}
}
