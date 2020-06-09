package main

import (
  "testing"

  . "gopkg.in/check.v1"
  )

func TestBalancer(t *testing.T) { TesingT(t) }

type MySuite struct{}

func (s *MySuite) TestLoadBalancer(c *C) {
  serverConnection[0] = 30
  serverConnection[1] = 20
  serverConnection[2] = 10

  var balancedServer1, _ = getServer()

  c.Assert(serverPool[2], Equals, balancedServer1)

  serverHealthStatus[0] = false
	serverConnection[1] = 200
	serverHealthStatus[2] = false

  var balancedServer2, _ = getServer()

  c.Assert(serverPool[1], Equals, balancedServer2)

  serverHealthStatus[0] = false
	serverHealthStatus[1] = false
	serverHealthStatus[2] = false

  var _, err = getServer()

  c.Assert("There is no healthy servers", Equals, err)
}
