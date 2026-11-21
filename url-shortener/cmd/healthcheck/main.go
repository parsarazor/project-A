package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// healthcheck performs an HTTP GET against the local /health endpoint and
// exits 0 only on a 2xx response. It is used as the Docker HEALTHCHECK for
// the scratch-based production image, which ships without a shell or wget.
func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%s/health", port)

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintf(os.Stderr, "healthcheck failed: unexpected status %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
