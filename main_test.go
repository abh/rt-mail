package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddressQueueMap(t *testing.T) {
	queue, action := addressToQueueAction("ntppool-support@rt.develooper.com")
	assert.Equal(t, "servers", queue)
	assert.Equal(t, "correspond", action)

	queue, action = addressToQueueAction("vendors-comment@rt.develooper.com")
	assert.Equal(t, "vendors", queue)
	assert.Equal(t, "comment", action)
}
