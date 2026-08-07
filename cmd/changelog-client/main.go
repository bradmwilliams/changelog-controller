package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	var (
		rcURL        string
		changelogURL string
		port         int
		timeout      time.Duration
	)

	flag.StringVar(&rcURL, "release-controller-url", "https://amd64.ocp.releases.ci.openshift.org",
		"Base URL of the release controller API")
	flag.StringVar(&changelogURL, "changelog-controller-url", "https://changelog.apps.engineering.openshift.org",
		"Base URL of the changelog controller API")
	flag.IntVar(&port, "port", 8080, "HTTP server port")
	flag.DurationVar(&timeout, "request-timeout", 30*time.Second,
		"Timeout for upstream API requests")
	flag.Parse()

	client := NewClient(rcURL, changelogURL, timeout)

	s := &Server{
		client:  client,
		streams: defaultStreams,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/releasestream/", s.handleStream)
	mux.HandleFunc("/", s.handleIndex)

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	log.Printf("changelog-client listening on %s", listener.Addr())
	log.Printf("  release-controller: %s", rcURL)
	log.Printf("  changelog-controller: %s", changelogURL)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
