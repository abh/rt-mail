package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockRTClient implements rt.Client for testing
type MockRTClient struct {
	PostmailFunc func(recipient string, message string) error
}

func (m *MockRTClient) Postmail(recipient string, message string) error {
	if m.PostmailFunc != nil {
		return m.PostmailFunc(recipient, message)
	}
	return nil
}

// NewMockRTServer creates a test HTTP server that simulates RT behavior
func NewMockRTServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		queue := r.FormValue("queue")
		if queue == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("failure: no queue specified"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("RT/4.4.4 200 Ok\n\n# Ticket 123 created."))
	}))
}

// AssertStatusCode checks the HTTP status code
func AssertStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status code = %d, want %d", got, want)
	}
}

// AssertNoError fails if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
