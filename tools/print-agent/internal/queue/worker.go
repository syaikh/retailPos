package queue

import (
	"context"
	"log"

	"print-agent/internal/receipt"
	"print-agent/internal/transport"
)

// RenderFunc renders a receipt to ESC/POS bytes.
type RenderFunc func(receipt.Receipt, receipt.Branding) ([]byte, error)

// RenderReturnFunc renders a return receipt to ESC/POS bytes.
type RenderReturnFunc func(receipt.ReturnReceipt, receipt.Branding) ([]byte, error)

// RenderDrawerFunc returns the ESC/POS bytes that pulse the cash drawer. It
// takes no arguments because a pulse has no payload: the drawer hangs off the
// printer, not off the receipt, and everything that varies about it (pin,
// width) is the agent's own configuration.
type RenderDrawerFunc func() []byte

// Worker consumes jobs from the store and dispatches them via the transport.
type Worker struct {
	store        *Store
	render       RenderFunc
	renderReturn RenderReturnFunc
	renderDrawer RenderDrawerFunc
	trans        transport.Transport
	logf         func(string, ...interface{})
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

// SetReturnRenderer sets the return-receipt render function (see SetRenderer).
func (w *Worker) SetReturnRenderer(fn RenderReturnFunc) { w.renderReturn = fn }

// SetDrawerRenderer sets the cash-drawer pulse function (see SetRenderer).
func (w *Worker) SetDrawerRenderer(fn RenderDrawerFunc) { w.renderDrawer = fn }

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
	var data []byte
	var err error
	switch j.Type {
	case TypeDrawer:
		// A pulse is not a receipt: it must not reach the sale renderer, which
		// would reject it on the missing invoice and report a print failure
		// for a drawer that opened fine (or never had a chance to).
		if w.renderDrawer == nil {
			w.store.Update(id, StatusFailed, "no drawer renderer configured")
			return
		}
		data = w.renderDrawer()
	case TypeReturn:
		if w.renderReturn == nil {
			w.store.Update(id, StatusFailed, "no return renderer configured")
			return
		}
		data, err = w.renderReturn(j.Return, j.Branding)
	default:
		// TypeSale and anything unrecognized: rendering it as a sale is what
		// the agent always did, so a client that sends no type still prints.
		if w.render == nil {
			w.store.Update(id, StatusFailed, "no renderer configured")
			return
		}
		data, err = w.render(j.Receipt, j.Branding)
	}
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
