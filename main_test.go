package main

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type MainSuite struct {
}

var _ = Suite(&MainSuite{})

func (s *MainSuite) TestCompile(c *C) {
	// nop
}

func (s *MainSuite) TestAddressQueueMap(c *C) {

	queue, action := addressToQueueAction("ntppool-support@rt.develooper.com")
	c.Check(queue, Equals, "ntppool-support")
	c.Check(action, Equals, "correspond")

	queue, action = addressToQueueAction("vendors-comment@rt.develooper.com")
	c.Check(queue, Equals, "ntppool-vendors")
	c.Check(action, Equals, "comment")

}
