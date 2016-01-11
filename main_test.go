package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddressQueueMap(t *testing.T) {

	err := loadConfig("mandrill-rt.json.sample")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	var queue, action string

	queue, action = addressToQueueAction("example-support@rt.example")
	assert.Equal(t, "support", queue)
	assert.Equal(t, "correspond", action)

	queue, action = addressToQueueAction("support-help-comment@rt.example")
	assert.Equal(t, "support", queue)
	assert.Equal(t, "comment", action)

	queue, action = addressToQueueAction("help@rt.example")
	assert.Equal(t, "help", queue)
	assert.Equal(t, "correspond", action)

	queue, action = addressToQueueAction("help@example.com")
	assert.Equal(t, "example", queue)
	assert.Equal(t, "correspond", action)

	queue, action = addressToQueueAction("help-comment@example.com")
	assert.Equal(t, "example", queue)
	assert.Equal(t, "comment", action)
}
