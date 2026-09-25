package monitor

import (
	"errors"
	"fmt"
	"maps"
	"sync"
)

type CollectorStatusStore struct {
	statuses map[string]CollectorStatus
	mu       sync.RWMutex
}

func NewStatusesStore() *CollectorStatusStore {
	return &CollectorStatusStore{
		statuses: map[string]CollectorStatus{
			"cpu": {
				Required: true,
				State:    CollectorUnknown,
			},
			"memory": {
				Required: true,
				State:    CollectorUnknown,
			},
			"network": {
				Required: true,
				State:    CollectorUnknown,
			},
			"temperature": {
				Required: false,
				State:    CollectorUnknown,
			},
		},
	}
}

func (c *CollectorStatusStore) Set(collector string, result CollectorStatus) error {
	if err := validateNameCollector(collector); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses[collector] = result

	return nil
}

func (c *CollectorStatusStore) SetState(collector string, state CollectorState) error {
	if err := validateNameCollector(collector); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	status, ok := c.statuses[collector]

	if !ok {
		return fmt.Errorf("invalid collector name: %s", collector)
	}

	status.State = state
	c.statuses[collector] = status

	return nil
}

func (c *CollectorStatusStore) Get(collector string) (CollectorStatus, error) {
	if err := validateNameCollector(collector); err != nil {
		return CollectorStatus{}, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	status, ok := c.statuses[collector]

	if !ok {
		return CollectorStatus{}, fmt.Errorf("invalid collector name: %s", collector)
	}

	return status, nil
}

func (c *CollectorStatusStore) GetAll() map[string]CollectorStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make(map[string]CollectorStatus, len(c.statuses))

	maps.Copy(statuses, c.statuses)

	return statuses
}

func (s *CollectorStatusStore) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, status := range s.statuses {
		if status.Required && status.State != CollectorAvailable {
			return false
		}
	}

	return true
}

func validateNameCollector(collector string) error {
	if collector == "" {
		return errors.New("collector name cannot be empty")
	}

	switch collector {
	case "cpu", "memory", "network", "temperature":
		// name valid
	default:
		return fmt.Errorf("invalid collector name %s", collector)
	}

	return nil
}
