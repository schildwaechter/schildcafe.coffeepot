package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMachine_StartJob(t *testing.T) {
	m := NewMachine()

	// Test valid job
	req := StartJobRequest{
		Product: ProductCoffee,
	}
	job, err := m.StartJob(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if job == nil {
		t.Fatal("Expected job, got nil")
	}
	if job.Product != ProductCoffee {
		t.Errorf("Expected product %s, got %s", ProductCoffee, job.Product)
	}
	if job.JobID == "" {
		t.Error("Expected job ID to be set")
	}
	if m.GetState() != StateBrewing {
		t.Errorf("Expected state %d, got %d", StateBrewing, m.GetState())
	}

	// Test invalid product
	req2 := StartJobRequest{
		Product: Product("INVALID"),
	}
	_, err = m.StartJob(req2)
	if err == nil {
		t.Error("Expected error for invalid product")
	}

	// Test machine not available
	_, err = m.StartJob(req)
	if err == nil {
		t.Error("Expected error when machine is busy")
	}
}

func TestMachine_RetrieveJob(t *testing.T) {
	m := NewMachine()

	// Start a job
	req := StartJobRequest{
		Product: ProductEspresso,
	}
	job, err := m.StartJob(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Try to retrieve before ready (should fail)
	_, err = m.RetrieveJob(job.JobID)
	if err == nil {
		t.Error("Expected error when job is not ready")
	}

	// Wait for job to be ready
	time.Sleep(time.Until(job.JobReady) + 100*time.Millisecond)

	// Retrieve the job
	retrievedJob, err := m.RetrieveJob(job.JobID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if retrievedJob.JobRetrieved == nil {
		t.Error("Expected jobRetrieved to be set")
	}

	// Try to retrieve again (should fail)
	_, err = m.RetrieveJob(job.JobID)
	if err == nil {
		t.Error("Expected error when job already retrieved")
	}

	// Test non-existent job
	_, err = m.RetrieveJob("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestMachine_GetAllJobs(t *testing.T) {
	m := NewMachine()

	// Start multiple jobs
	for i := 0; i < 3; i++ {
		req := StartJobRequest{
			Product: ProductCoffee,
		}
		_, err := m.StartJob(req)
		if err == nil {
			// Wait for job to complete
			time.Sleep(100 * time.Millisecond)
			// Retrieve to make machine available
			jobs := m.GetAllJobs()
			if len(jobs) > 0 {
				time.Sleep(time.Until(jobs[len(jobs)-1].JobReady) + 100*time.Millisecond)
				m.RetrieveJob(jobs[len(jobs)-1].JobID)
			}
		}
	}

	jobs := m.GetAllJobs()
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestHandleStartJob(t *testing.T) {
	machine = NewMachine()

	reqBody := `{"product": "COFFEE"}`
	req := httptest.NewRequest(http.MethodPost, "/start-job", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleStartJob(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var job Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if job.Product != ProductCoffee {
		t.Errorf("Expected product COFFEE, got %s", job.Product)
	}
}

func TestHandleRetrieveJob(t *testing.T) {
	machine = NewMachine()

	// Start a job
	reqBody := `{"product": "ESPRESSO"}`
	req := httptest.NewRequest(http.MethodPost, "/start-job", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleStartJob(w, req)

	var job Job
	json.NewDecoder(w.Body).Decode(&job)

	// Wait for job to be ready
	time.Sleep(time.Until(job.JobReady) + 100*time.Millisecond)

	// Retrieve the job
	req2 := httptest.NewRequest(http.MethodGet, "/retrieve-job?jobID="+job.JobID, nil)
	w2 := httptest.NewRecorder()
	handleRetrieveJob(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}
}

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handleHealthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestHandleReadyz(t *testing.T) {
	machine = NewMachine()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handleReadyz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	machine = NewMachine()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	machine = NewMachine()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "coffee_machine_status") {
		t.Error("Expected metrics to contain coffee_machine_status")
	}
}

func TestHandleHistory(t *testing.T) {
	machine = NewMachine()

	// Start a job
	reqBody := `{"product": "CAPPUCCINO"}`
	req := httptest.NewRequest(http.MethodPost, "/start-job", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleStartJob(w, req)

	// Get history
	req2 := httptest.NewRequest(http.MethodGet, "/history", nil)
	w2 := httptest.NewRecorder()
	handleHistory(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	var jobs []Job
	if err := json.NewDecoder(w2.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(jobs) == 0 {
		t.Error("Expected at least one job in history")
	}
}

func TestValidProducts(t *testing.T) {
	validProducts := []Product{
		ProductCoffee,
		ProductStrongCoffee,
		ProductCappuccino,
		ProductCoffeeWithMilk,
		ProductEspresso,
		ProductEspressoChocolate,
		ProductKakao,
		ProductHotWater,
	}

	for _, product := range validProducts {
		if !ValidProducts[product] {
			t.Errorf("Product %s should be valid", product)
		}
	}

	invalidProduct := Product("INVALID")
	if ValidProducts[invalidProduct] {
		t.Errorf("Product %s should not be valid", invalidProduct)
	}
}
