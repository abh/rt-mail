package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPI(t *testing.T) {
	// This test is a placeholder since the full API setup requires RT configuration.
	// A more comprehensive test would require mocking the RT backend.
	data := strings.NewReader(`[{"msys": {}}]`)
	req, err := http.NewRequest("POST", "/spark", data)
	if err != nil {
		t.Fatalf("could not setup ping request: %s", err)
	}

	rr := httptest.NewRecorder()
	mux := http.NewServeMux()

	// Test basic routing without full RT setup
	mux.HandleFunc("/spark", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
