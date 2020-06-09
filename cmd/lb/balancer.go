package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"net/http"
	"time"
	"errors"

	"github.com/SergeyStrashko/design-practice-3/httptools"
	"github.com/SergeyStrashko/design-practice-3/signal"
)

var (
	port = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout = time.Duration(*timeoutSec) * time.Second
	serverPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}

	serverTraffic = []int64{
		0,
		0,
		0,
	}

	serverConnection = []int64{
		0,
		0,
		0,
	}

	serverHealthStatus = []bool {
		true,
		true,
		true,
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string, i int) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	traffic, err := strconv.ParseInt(string(buf.Bytes()), 10, 64)

	if err != nil {
		return false
	}
	log.Printf("Response health %s: %d", dst, traffic)

	if resp.StatusCode != http.StatusOK {
		return false
	}

	serverTraffic[i] = traffic
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request, index int64) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}

		serverConnection[index] += 1

		return nil
	} else {
		serverConnection[index] -= 1

		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func getServer() (string, error) {
	var serverIndex = 0;

	for index := 0; index < 3; index++ {
		log.Printf("Connections of server %d: %d", i, serverTraffic[index])
		if serverHealthStatus[index] {
			if serverConnection[index] < serverConnection[serverIndex] {
				serverIndex = index
			}
		}
	}

	if !serverHealthStatus[serverIndex] {
		return serverPool[0], errors.errors.New("There is no healthy servers")
	}

	return serverPool[serverIndex], nil
}

func main() {
	flag.Parse()

	go func() {
		for range time.Tick(10 * time.Hour) {
			serverTraffic[0] = 0
			serverTraffic[1] = 0
			serverTraffic[2] = 0

			serverConnection[0] = 0
			serverConnection[1] = 0
			serverConnection[2] = 0
		}
	}()

	for index := 0; index < 3; index++ {
		server := serversPool[index]
		go func() {
			for range time.Tick(10 * time.Second) {
				serverHealthStatus[index] = health(server, index)
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var Server, err = getServer()
		if err == nil {
			log.Printf("Forwarding to server: %s", Server.url)
			forward(&Server, rw, r, index)
		} else {
			log.Printf("Request error: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
