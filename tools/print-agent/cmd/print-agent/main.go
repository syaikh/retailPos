// Command print-agent is the local print agent for the POS. It receives receipt
// payloads from the browser and dispatches them to a printer (file/tcp/serial)
// as ESC/POS bytes. It is a standalone, dependency-free Go binary intended to run
// as a per-terminal service (systemd / Windows service).
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[print-agent] invalid configuration: %v", err)
	}

	// A stale PRINT_TOKEN in an env file is ignored rather than fatal, because
	// refusing to start would take every till's printer offline for a setting
	// that no longer does anything. Warning instead means the leftover is
	// visible in the log without stopping the agent.
	if os.Getenv("PRINT_TOKEN") != "" {
		log.Println("[print-agent] warning: PRINT_TOKEN is set but no longer used; " +
			"bearer auth was removed because the only client is a browser. " +
			"Delete it from the env file. Access is controlled by ALLOWED_ORIGINS.")
	}

	trans, err := transport.New(transport.Config{
		Kind:         cfg.Transport,
		OutputDir:    cfg.OutputDir,
		TCPAddr:      cfg.TCPAddr,
		SerialDevice: cfg.SerialDevice,
	})
	if err != nil {
		log.Fatalf("[print-agent] transport init failed: %v", err)
	}
	defer func() { _ = trans.Close() }()

	store := queue.NewStore()
	pm := printer.New(trans)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker := queue.NewWorker(store, trans)
	worker.SetRenderer(receipt.Render)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Run(ctx)
	}()

	mux := http.NewServeMux()
	h := api.NewHandler(store, pm, cfg)
	h.Register(mux)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.BindAddress, cfg.Port),
		Handler:           h.Middleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// The agent holds no credential. Logging that explicitly is useful: a
	// startup line that lists the bind address, transport and the exact origins
	// it will accept is how an operator confirms a register is pointed at the
	// right agent and that no other origin is allowed.
	log.Printf("[print-agent] listening on %s transport=%s origins=%v auth=origin-allowlist",
		srv.Addr, cfg.Transport, cfg.AllowedOrigins)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[print-agent] server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("[print-agent] shutting down, draining queued jobs")

	// Stop accepting new HTTP requests. In-flight requests are allowed to finish
	// (and may still enqueue jobs), so wait for them before draining.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[print-agent] server shutdown error: %v", err)
	}
	shutdownCancel()

	// No new jobs can arrive now; let the worker drain the queue (including any
	// job currently printing) before exiting. Bound the wait so a hung transport
	// cannot block shutdown forever.
	cancel()

	drainDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(drainDone)
	}()
	select {
	case <-drainDone:
	case <-time.After(15 * time.Second):
		log.Println("[print-agent] drain timed out; forcing exit")
	}

	log.Println("[print-agent] stopped")
}
