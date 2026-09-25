package metrics

import (
	"errors"
	"fmt"
	"regexp"
)

func validateMetric(result MetricResult) error {
	if result.Name == "" {
		return errors.New("metric name cannot be empty")
	}

	switch result.Type {
	case "gauge", "counter":
		// valid
	default:
		return fmt.Errorf("invalid metric type %q", result.Type)
	}

	for labelName := range result.Labels {
		if !isValidLabelName(labelName) {
			return fmt.Errorf("invalid label name %q", labelName)
		}
	}

	return nil
}

var labelNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidLabelName(name string) bool {
	return labelNameRegex.MatchString(name)
}
