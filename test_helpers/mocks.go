package test_helpers

import (
	"log"
	"net"
	"net/http"
)

// Spin up a simple file server for serving sitemaps or other static files
func ServerForStaticFile() (*http.Server, net.Listener, error) {

	http.Handle("/", http.FileServer(http.Dir("./sample_configs")))

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}

	// Start the server
	server := &http.Server{
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Printf("Static file server running on %s", listener.Addr().String())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Error serving: %v", err)
		}
	}()

	return server, listener, nil
}
