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
	recorded.CodeIs(204)
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

func TestAddressQueueMap(t *testing.T) {

	err := loadConfig("sparkpost-rt.json.sample")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	tests := [][]string{
		[]string{"example-support@rt.example", "support", "correspond"},
		[]string{"support-help-comment@rt.example", "support", "comment"},
		[]string{"help@rt.example", "help", "correspond"},
		[]string{"help@example.com", "example", "correspond"},
		[]string{"help-comment@example.com", "example", "comment"},
	}

	for _, test := range tests {
		queue, action := addressToQueueAction(test[0])
		if queue != test[1] {
			t.Logf("testing '%s' got queue '%s' but expected '%s'", test[0], queue, test[1])
			t.Fail()
		}
		if action != test[2] {
			t.Logf("testing '%s' got action '%s' but expected '%s'", test[0], action, test[2])
			t.Fail()
		}

	}
}
