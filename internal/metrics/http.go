package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func NewMetricsHandler(store *MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		metrics := store.GetAll()
		groupedMetrics := groupMetrics(metrics)

		// Obtener y ordenar nombres de métricas.
		names := make([]string, 0, len(groupedMetrics))

		for name := range groupedMetrics {
			names = append(names, name)
		}
		sort.Strings(names)

		// Procesar cada grupo en orden.
		for _, name := range names {
			results := groupedMetrics[name]

			if len(results) == 0 {
				continue
			}

			// Ordenar muestras por sus labels.
			sort.SliceStable(results, func(i, j int) bool {
				return labelsKey(results[i].Labels) <
					labelsKey(results[j].Labels)
			})

			first := results[0]

			fmt.Fprintf(
				w,
				"# HELP %s %s\n",
				name,
				first.Description,
			)

			fmt.Fprintf(
				w,
				"# TYPE %s %s\n",
				name,
				first.Type,
			)

			for _, result := range results {
				fmt.Fprintln(w, formatMetric(result))
			}

			fmt.Fprintln(w)
		}
	}
}

func groupMetrics(metrics map[string]MetricResult) map[string][]MetricResult {
	grouped := make(map[string][]MetricResult)

	for _, metric := range metrics {
		grouped[metric.Name] = append(grouped[metric.Name], metric)
	}

	return grouped
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var b strings.Builder

	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(labels[key])
		b.WriteByte(';')
	}

	return b.String()
}
