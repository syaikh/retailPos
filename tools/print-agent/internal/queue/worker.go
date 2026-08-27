package queue

import (
	"context"
	"log"

	"print-agent/internal/receipt"
	"print-agent/internal/transport"
)

// RenderFunc renders a receipt to ESC/POS bytes.
type RenderFunc func(receipt.Receipt, receipt.Branding) ([]byte, error)

// Worker consumes jobs from the store and dispatches them via the transport.
type Worker struct {
	store  *Store
	render RenderFunc
	trans  transport.Transport
	logf   func(string, ...interface{})
}

// NewWorker builds a worker. The render function must be set (defaults to
// receipt.Render if left nil).
func NewWorker(store *Store, trans transport.Transport) *Worker {
	return &Worker{
		store: store,
		trans: trans,
		logf:  log.Printf,
	}
}

// SetRenderer sets the render function (used so the worker package need not
// hard-depend on the receipt package at construction time).
func (w *Worker) SetRenderer(fn RenderFunc) { w.render = fn }

// Run processes jobs until the context is cancelled, then drains any remaining
// queued jobs before returning so in-flight and pending prints are not lost on
// shutdown.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.drain()
			return
		case id := <-w.store.Channel():
			w.process(id)
		}
	}
}

// drain processes any jobs still buffered on the channel. It returns once the
// channel is empty; the caller must have already stopped accepting new network
// requests so no further Enqueue can occur.
func (w *Worker) drain() {
	for {
		select {
		case id := <-w.store.Channel():
			w.process(id)
		default:
			return
		}
	}
}

func (w *Worker) process(id string) {
	j, ok := w.store.Get(id)
	if !ok || j.Status != StatusQueued {
		return
	}
	w.store.Update(id, StatusPrinting, "")
	if w.render == nil {
		w.store.Update(id, StatusFailed, "no renderer configured")
		return
	}
	data, err := w.render(j.Receipt, j.Branding)
	if err != nil {
		w.store.Update(id, StatusFailed, err.Error())
		return
	}
	if err := w.trans.Write(id, data); err != nil {
		w.store.Update(id, StatusFailed, err.Error())
		w.logf("[print-agent] job %s failed: %v", id, err)
		return
	}
	w.store.Update(id, StatusCompleted, "")
	w.logf("[print-agent] job %s completed", id)
}
