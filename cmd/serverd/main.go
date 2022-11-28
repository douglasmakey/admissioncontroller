package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/douglasmakey/admissioncontroller/http"

	log "k8s.io/klog/v2"
)

var (
	tlscert, tlskey, port string
)

func main() {
	flag.StringVar(&tlscert, "tlscert", "/etc/certs/tls.crt", "Path to the TLS certificate")
	flag.StringVar(&tlskey, "tlskey", "/etc/certs/tls.key", "Path to the TLS key")
	flag.StringVar(&port, "port", "8443", "The port on which to listen")
	flag.Parse()

	server := http.NewServer(port)

	go func() {
		// listen shutdown signal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan
		log.Errorf("Received %s signal; shutting down...", sig)
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error(err)
		}
	}()

	log.Infof("Starting server on port: %s", port)
	if err := server.ListenAndServeTLS(tlscert, tlskey); err != nil {
		log.Errorf("Failed to listen and serve: %v", err)
		os.Exit(1)
	}
}
