package main

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestWaitForSetupCompletionStopsSetupServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "setup")
		}),
	}
	t.Cleanup(func() { _ = server.Close() })
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	client := &http.Client{Timeout: time.Second}
	url := "http://" + listener.Addr().String()
	response, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET setup server error = %v", err)
	}
	_ = response.Body.Close()

	completed := make(chan struct{})
	close(completed)

	if !waitForSetupCompletion(server, completed, serverErr) {
		t.Fatal("waitForSetupCompletion() = false, want true")
	}
	if _, err := client.Get(url); err == nil {
		t.Fatal("setup server still accepts requests after completion")
	}
}
