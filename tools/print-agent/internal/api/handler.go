// Package api implements the print agent HTTP API.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"print-agent/internal/config"
	"print-agent/internal/printer"
	"print-agent/internal/queue"
	"print-agent/internal/receipt"
)

// Handler serves the print agent endpoints.
type Handler struct {
	store   *queue.Store
	printer *printer.Manager
	cfg     config.Config
}

// NewHandler constructs the API handler.
func NewHandler(store *queue.Store, pm *printer.Manager, cfg config.Config) *Handler {
	return &Handler{store: store, printer: pm, cfg: cfg}
}

// Middleware applies CORS, optional bearer-token auth, and OPTIONS handling.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allow := h.allowedOrigin(origin)
		w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if h.cfg.Token != "" {
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if tok != h.cfg.Token {
				writeJSON(w, http.StatusUnauthorized, errBody("unauthorized"))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) allowedOrigin(origin string) string {
	if origin == "" {
		return "*"
	}
	// Default (no configured origins) reflects the request origin, which allows
	// any origin while keeping the header specific. Restrict via ALLOWED_ORIGINS
	// in production.
	if len(h.cfg.AllowedOrigins) == 0 {
		return origin
	}
	for _, o := range h.cfg.AllowedOrigins {
		if o == "*" || o == origin {
			return origin
		}
	}
	return "null"
}

// Register wires the routes onto the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/printer", h.handlePrinter)
	mux.HandleFunc("/print", h.handlePrint)
	mux.HandleFunc("/print/jobs/", h.handleJob)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	connected, kind := h.printer.Status()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
		"printer": map[string]interface{}{
			"connected": connected,
			"type":      kind,
		},
	})
}

func (h *Handler) handlePrinter(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	connected, kind := h.printer.Status()
	status := "ready"
	if !connected {
		status = "unavailable"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":     "receipt",
		"name":   "Thermal Printer",
		"type":   kind,
		"status": status,
	})
}

func (h *Handler) handlePrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
		return
	}
	var req receipt.PrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("invalid json: "+err.Error()))
		return
	}

	rc := req.Data
	if rc.InvoiceNumber == "" {
		rc = req.Receipt
	}
	if rc.InvoiceNumber == "" {
		writeJSON(w, http.StatusBadRequest, errBody("missing receipt data"))
		return
	}

	jobID := req.JobID
	if jobID == "" {
		jobID = "print-" + randHex(10)
	}

	// Idempotency: the same job_id returns the existing job instead of reprinting.
	if existing, ok := h.store.Get(jobID); ok {
		writeJSON(w, http.StatusOK, map[string]string{
			"job_id": jobID,
			"status": string(existing.Status),
		})
		return
	}

	h.store.Enqueue(&queue.Job{
		ID:       jobID,
		Receipt:  rc,
		Branding: req.Branding,
	})
	writeJSON(w, http.StatusAccepted, map[string]string{
		"job_id": jobID,
		"status": string(queue.StatusQueued),
	})
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"jobs": h.store.List()})
}

func (h *Handler) handleJob(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/print/jobs/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		h.listJobs(w, r)
		return
	}

	job, ok := h.store.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("job not found"))
		return
	}

	switch {
	case r.Method == "GET":
		writeJSON(w, http.StatusOK, map[string]string{
			"job_id": id,
			"status": string(job.Status),
			"error":  job.Error,
		})
	case r.Method == "POST" && len(parts) > 1 && parts[1] == "retry":
		if job.Status != queue.StatusFailed {
			writeJSON(w, http.StatusConflict, errBody("job not retryable"))
			return
		}
		h.store.Update(id, queue.StatusQueued, "")
		h.store.Requeue(id)
		writeJSON(w, http.StatusOK, map[string]string{
			"job_id": id,
			"status": string(queue.StatusQueued),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("[print-agent] encode error: %v", err)
	}
}

func errBody(msg string) map[string]string {
	return map[string]string{"ok": "false", "error": msg}
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail; fall back to timestamp-based id.
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}
