package main

import (
	"io/ioutil"
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
		t.Fatalf("Could not setup ping request: %s", err)
	}
	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(200)
}

func TestAPIMX(t *testing.T) {
	return
	api := newAPI()

	rawmsg, err := ioutil.ReadFile("msg.json")
	if err != nil {
		t.Fatalf("Could not read 'msg.json' test data: %s", err)
	}

	req := test.MakeSimpleRequest("POST", "/spark/mx", rawmsg)
	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)
}
