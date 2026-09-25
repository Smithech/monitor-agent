package config

import (
	"fmt"
	"time"
)

type Config struct {
	CPUInterval         time.Duration
	MemoryInterval      time.Duration
	NetworkInterval     time.Duration
	TemperatureInterval time.Duration
	RetryInterval       time.Duration
}

func DefaultConfig() Config {
	return Config{
		CPUInterval:         3 * time.Second,
		MemoryInterval:      30 * time.Second,
		NetworkInterval:     30 * time.Second,
		TemperatureInterval: 30 * time.Second,
		RetryInterval:       1 * time.Second,
	}
}

func (c Config) Validate() error {
	if c.CPUInterval <= 0 {
		return fmt.Errorf("CPUInterval must be greater than zero")
	}

	if c.MemoryInterval <= 0 {
		return fmt.Errorf("MemoryInterval must be greater than zero")
	}

	if c.NetworkInterval <= 0 {
		return fmt.Errorf("NetworkInterval must be greater than zero")
	}

	if c.TemperatureInterval <= 0 {
		return fmt.Errorf("TemperatureInterval must be greater than zero")
	}

	if c.RetryInterval <= 0 {
		return fmt.Errorf("RetryInterval must be greater than zero")
	}

	return nil
}
