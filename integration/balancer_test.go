package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

var (
	m = map[string]int {
		"server1:8080": 0,
		"server2:8080": 0,
		"server3:8080": 0,
	}
	avgTime = [10]float64{}
)

type MySuite struct{}

func TestBalancer(t *testing.T) {
	finished := make(chan bool)
	go BenchmarkBalancer(t, finished, m)
	<- finished
	log.Println("tests are ok!")
}

func BenchmarkBalancer(c *C, finished chan bool, serverCounter map[string]int) {
	counter := 0
	for range time.Tick(11 * time.Second) {
		start := time.Now()
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Error(err)
			finished <- true
			return
		}
		duration := time.Since(start).Seconds()
		avgTime[counter] = duration

		serverCounter[resp.Header.Get("lb-from")] += 1
		t.Logf("response from [%s]", resp.Header.Get("lb-from"))
		counter += 1
		if counter == 10 {
			break
		}
	}
	log.Println(serverCounter["server1:8080"])
	c.Assert(serverCounter["server1:8080"] >= 3, Equals, true)
	log.Println(serverCounter["server2:8080"])
	c.Assert(serverCounter["server2:8080"] >= 3, Equals, true)
	log.Println(serverCounter["server3:8080"])
	c.Assert(serverCounter["server3:8080"] >= 3, Equals, true)

	var avg float64 = 0
	for i := 0; i < 10; i++ {
		avg += avgTime[i]
	}
	avg /= 10
	c.Assert(avg < client.Timeout.Seconds(), Equals, true)
	log.Println("Benchmark is ok")
	log.Printf("Benchmark avg: %g", avg)
	finished <- true
}
