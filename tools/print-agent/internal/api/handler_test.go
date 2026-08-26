package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	jobID, _ := created["job_id"].(string)
	if jobID == "" {
		t.Fatal("no job_id returned")
	}

	// Wait for the worker to complete the job.
	var status string
	for i := 0; i < 100; i++ {
		resp2, _ := http.Get(srv.URL + "/print/jobs/" + jobID)
		var js map[string]interface{}
		json.NewDecoder(resp2.Body).Decode(&js)
		resp2.Body.Close()
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
	resp3.Body.Close()
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
	resp1.Body.Close()

	resp2, _ := http.Post(srv.URL+"/print", "application/json", bytes.NewReader(payload()))
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("duplicate post expected 200, got %d", resp2.StatusCode)
	}
	var dup map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&dup)
	resp2.Body.Close()
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("missing CORS header: %v", resp.Header)
	}
}
