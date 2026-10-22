package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"print-agent/internal/config"
	"print-agent/internal/printer"
	"print-agent/internal/queue"
	"print-agent/internal/receipt"
	"print-agent/internal/transport"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	trans, err := transport.New(transport.Config{Kind: "file", OutputDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	store := queue.NewStore()
	pm := printer.New(trans)
	worker := queue.NewWorker(store, trans)
	worker.SetRenderer(receipt.Render)
	worker.SetReturnRenderer(receipt.RenderReturn)
	// main.go wires the real encoder here; the bytes are irrelevant to an
	// HTTP-level test, but something must be configured or a drawer job would
	// fail for a missing renderer rather than for anything under test.
	worker.SetDrawerRenderer(func() []byte { return []byte{0x1B, 'p', 0, 25} })
	go worker.Run(context.Background())

	h := NewHandler(store, pm, config.Config{})
	mux := http.NewServeMux()
	h.Register(mux)
	return httptest.NewServer(h.Middleware(mux))
}

func TestPrintLifecycle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"invoice_number": "INV-1",
			"items": []map[string]interface{}{
				{"name": "Kopi", "quantity": 2, "unit_price": 5000},
			},
			"total_amount": 10000,
			"subtotal_dpp": 9009,
			"tax":          991,
			"payments": []map[string]interface{}{
				{"method": "cash", "amount": 10000},
			},
		},
		"branding": map[string]interface{}{"storeName": "Toko Test"},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	var created map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response into created: %v", err)
	}
	_ = resp.Body.Close()
	jobID, _ := created["job_id"].(string)
	if jobID == "" {
		t.Fatal("no job_id returned")
	}

	// Wait for the worker to complete the job.
	var status string
	for i := 0; i < 100; i++ {
		resp2, _ := http.Get(srv.URL + "/print/jobs/" + jobID)
		var js map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&js); err != nil {
			t.Fatalf("decode response into js: %v", err)
		}
		_ = resp2.Body.Close()
		status, _ = js["status"].(string)
		if status == "completed" || status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status != "completed" {
		t.Fatalf("job did not complete, status=%s", status)
	}

	// Health endpoint.
	resp3, _ := http.Get(srv.URL + "/health")
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("health status %d", resp3.StatusCode)
	}
	_ = resp3.Body.Close()
}

func TestPrintIdempotency(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	payload := func() []byte {
		b, _ := json.Marshal(map[string]interface{}{
			"job_id": "print-fixed-id",
			"data":   map[string]interface{}{"invoice_number": "INV-9", "total_amount": 1000},
		})
		return b
	}

	resp1, _ := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("first post expected 202, got %d", resp1.StatusCode)
	}
	_ = resp1.Body.Close()

	resp2, _ := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("duplicate post expected 200, got %d", resp2.StatusCode)
	}
	var dup map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&dup); err != nil {
		t.Fatalf("decode response into dup: %v", err)
	}
	_ = resp2.Body.Close()
	if dup["status"] == nil {
		t.Fatal("duplicate response missing status")
	}
}

func TestCORSPreflight(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/print", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("missing CORS header: %v", resp.Header)
	}
}

// payload builds a minimal valid print request body.
func payload() []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"job_id": "print-test-id",
		"data":   map[string]interface{}{"invoice_number": "INV-TEST", "total_amount": 1000},
	})
	return b
}

// Bearer auth was removed. A token configured anywhere in the environment must
// no longer be able to reject a request, because the only client is a browser
// and would never send it, so an enabled token meant printing never worked.
func TestStaleTokenInEnvironmentDoesNotBlockPrinting(t *testing.T) {
	t.Setenv("PRINT_TOKEN", "leftover-from-an-old-env-file")
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("stale PRINT_TOKEN still rejects requests; bearer auth was meant to be gone")
	}
	// A brand new job id is accepted for printing, not replayed.
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
}

// Authorization is no longer advertised in the preflight response. Advertising a
// header the agent never reads invites a client to send a credential that buys
// nothing.
func TestPreflightDoesNotOfferAuthorizationHeader(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/print", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	allowed := resp.Header.Get("Access-Control-Allow-Headers")
	if strings.Contains(allowed, "Authorization") {
		t.Fatalf("preflight still offers Authorization in %q", allowed)
	}
}

func TestReturnPrintLifecycle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	payload := map[string]interface{}{
		"type": "return",
		"return_data": map[string]interface{}{
			"invoice_number":          "RETUR-1",
			"created_at":              "2026-08-26 11:00",
			"return_type":             "refund",
			"original_invoice_number": "INV-1",
			"refund_amount":           10000,
			"payments": []map[string]interface{}{
				{"method": "CASH", "amount": 10000},
			},
			"reason":    "Barang rusak",
			"condition": "Rusak",
			"items": []map[string]interface{}{
				{"name": "Kopi", "quantity": 2, "unit_price": 5000},
			},
		},
		"branding": map[string]interface{}{"storeName": "Toko Test"},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	var created map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response into created: %v", err)
	}
	jobID, _ := created["job_id"].(string)
	if jobID == "" {
		t.Fatal("no job_id returned")
	}

	// The return payload must reach the return renderer and complete — not be
	// dropped by the sale path (which would find no invoice and fail).
	var status string
	for i := 0; i < 100; i++ {
		resp2, _ := http.Get(srv.URL + "/print/jobs/" + jobID)
		var js map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&js); err != nil {
			t.Fatalf("decode response into js: %v", err)
		}
		_ = resp2.Body.Close()
		status, _ = js["status"].(string)
		if status == "completed" || status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status != "completed" {
		t.Fatalf("return job did not complete, status=%s", status)
	}
}

// A "return" request whose return_data is missing must be rejected outright:
// falling through to the sale path would either 400 confusingly or, worse,
// print whatever the sale slots held.
func TestReturnPrintMissingDataRejected(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "return",
		"return_data": map[string]interface{}{},
		"data":        map[string]interface{}{"invoice_number": "INV-SHOULD-NOT-PRINT"},
	})
	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a return request without return data, got %d", resp.StatusCode)
	}
}

// A drawer request carries no receipt at all: there is nothing to validate and
// nothing to render, only a pulse to put on the wire. It must still be accepted,
// dispatched through the queue and reach the printer like any other job — the
// drawer is only reachable through the printer's own transport.
func TestDrawerPrintLifecycle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{"type": "drawer"})
	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for a payload-free drawer request, got %d", resp.StatusCode)
	}

	var created map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response into created: %v", err)
	}
	jobID, _ := created["job_id"].(string)
	if jobID == "" {
		t.Fatal("no job_id returned")
	}
	// The drawer prefix is what an operator reads out of the log when a till
	// will not open, so it has to survive into the API response.
	if !strings.HasPrefix(jobID, "drawer-") {
		t.Fatalf("job_id %q, want a drawer- prefix", jobID)
	}

	var status string
	for i := 0; i < 100; i++ {
		resp2, _ := http.Get(srv.URL + "/print/jobs/" + jobID)
		var js map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&js); err != nil {
			t.Fatalf("decode response into js: %v", err)
		}
		_ = resp2.Body.Close()
		status, _ = js["status"].(string)
		if status == "completed" || status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status != "completed" {
		t.Fatalf("drawer job did not complete, status=%s", status)
	}
}

// The drawer inherits job-id dedupe rather than getting its own write path: a
// replayed POST (a proxy retry, a double submit) must return the existing job
// instead of pulsing the till a second time.
func TestDrawerPrintIdempotency(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	payload := func() []byte {
		b, _ := json.Marshal(map[string]interface{}{
			"job_id": "drawer-fixed-id",
			"type":   "drawer",
		})
		return b
	}

	resp1, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if err != nil {
		t.Fatal(err)
	}
	if resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("first post expected 202, got %d", resp1.StatusCode)
	}
	_ = resp1.Body.Close()

	resp2, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("duplicate post expected 200, got %d", resp2.StatusCode)
	}
	var dup map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&dup); err != nil {
		t.Fatalf("decode response into dup: %v", err)
	}
	if dup["job_id"] != "drawer-fixed-id" {
		t.Fatalf("duplicate job_id = %q, want %q", dup["job_id"], "drawer-fixed-id")
	}
	if dup["status"] == "" {
		t.Fatal("duplicate response missing status")
	}
}

// A type this agent does not recognise falls through to the sale path — the
// same code an agent predating a newer job type runs when handed that type.
// With no receipt attached it must refuse the request rather than accept work
// it cannot serve.
func TestPrintUnknownTypeWithoutReceiptRejected(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{"type": "drawer-v2"})
	resp, err := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown type without receipt data, got %d", resp.StatusCode)
	}
	var errResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if !strings.Contains(errResp["error"], "missing receipt data") {
		t.Fatalf("error = %q, want it to contain %q", errResp["error"], "missing receipt data")
	}
}
