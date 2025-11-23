package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHandleRetrieveJobUsesJobIDParameter(t *testing.T) {
	m := NewMachine()
	m.brewTimeFn = func() time.Duration { return time.Millisecond }

	job, err := m.StartJob(ProductEspresso, "")
	if err != nil {
		t.Fatalf("start job: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	handler := newServer(m, log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodGet, "/retrieve-job?jobID="+url.QueryEscape(job.JobID), nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusOK)
	}

	var got Job
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.JobID != job.JobID {
		t.Fatalf("expected job %s, got %s", job.JobID, got.JobID)
	}
}

func TestHandleRetrieveJobMissingJobID(t *testing.T) {
	handler := newServer(NewMachine(), log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodGet, "/retrieve-job?jobId=legacy", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if body := rec.Body.String(); body != "missing jobID\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}
