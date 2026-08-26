// Command print-agent is the local print agent for the POS. It receives receipt
// payloads from the browser and dispatches them to a printer (file/tcp/serial)
// as ESC/POS bytes. It is a standalone, dependency-free Go binary intended to run
// as a per-terminal service (systemd / Windows service).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"print-agent/internal/api"
	"print-agent/internal/config"
	"print-agent/internal/printer"
	"print-agent/internal/queue"
	"print-agent/internal/receipt"
	"print-agent/internal/transport"
)

func main() {
	cfg := config.Load()

	trans, err := transport.New(transport.Config{
		Kind:         cfg.Transport,
		OutputDir:    cfg.OutputDir,
		TCPAddr:      cfg.TCPAddr,
		SerialDevice: cfg.SerialDevice,
	})
	if err != nil {
		log.Fatalf("[print-agent] transport init failed: %v", err)
	}
	defer trans.Close()

	store := queue.NewStore()
	pm := printer.New(trans)

	worker := queue.NewWorker(store, trans)
	worker.SetRenderer(receipt.Render)
	go worker.Run(context.Background())

	mux := http.NewServeMux()
	h := api.NewHandler(store, pm, cfg)
	h.Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Middleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("[print-agent] listening on :%s transport=%s", cfg.Port, cfg.Transport)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[print-agent] server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("[print-agent] shutting down")
}
