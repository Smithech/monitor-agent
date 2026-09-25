package metrics

type MetricResult struct {
	Name        string
	Description string
	Type        string
	Value       float64
	Labels      map[string]string
}
