package metrics

import (
	"fmt"
	"net/http"
)

type ReadinessChecker interface {
	Ready() bool
}

func ReadyHandler(checker ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if !checker.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "NOT READY")
			return
		}

		fmt.Fprintln(w, "READY")
	}
}
