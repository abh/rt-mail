package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestAPI(t *testing.T) {
	api := newAPI()

	data := strings.NewReader(`[{"msys": {}}]`)
	req, err := http.NewRequest("POST", "/spark", data)
	if err != nil {
		t.Fatalf("could not setup ping request: %s", err)
	}
	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(200)
}
