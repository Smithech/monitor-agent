package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeReadinessChecker struct {
	ready bool
}

func (f fakeReadinessChecker) Ready() bool {
	return f.ready
}

func TestReadyHandler(t *testing.T) {
	tests := []struct {
		name           string
		ready          bool
		method         string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "ready",
			ready:          true,
			method:         http.MethodGet,
			wantStatusCode: http.StatusOK,
			wantBody:       "READY\n",
		},
		{
			name:           "not ready",
			ready:          false,
			method:         http.MethodGet,
			wantStatusCode: http.StatusServiceUnavailable,
			wantBody:       "NOT READY\n",
		},
		{
			name:           "método distinto de GET",
			ready:          true,
			method:         http.MethodPost,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := fakeReadinessChecker{
				ready: tt.ready,
			}

			req := httptest.NewRequest(tt.method, "/ready", nil)
			rec := httptest.NewRecorder()

			handler := ReadyHandler(checker)
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatusCode)
			}

			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
