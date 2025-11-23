package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	// Machine states
	StateAvailable = 0 // Available to receive a new job
	StateBrewing   = 1 // Brewing a job
	StateBlocked   = 2 // Blocked by a ready job that has not been retrieved

	// Brewing time range (in seconds)
	MinBrewingTime = 20
	MaxBrewingTime = 55
)

// Product represents a coffee machine product
type Product string

const (
	ProductCoffee            Product = "COFFEE"
	ProductStrongCoffee      Product = "STRONG_COFFEE"
	ProductCappuccino        Product = "CAPPUCCINO"
	ProductCoffeeWithMilk    Product = "COFFEE_WITH_MILK"
	ProductEspresso          Product = "ESPRESSO"
	ProductEspressoChocolate Product = "ESPRESSO_CHOCOLATE"
	ProductKakao             Product = "KAKAO"
	ProductHotWater          Product = "HOT_WATER"
)

// ValidProducts contains all supported products
var ValidProducts = map[Product]bool{
	ProductCoffee:            true,
	ProductStrongCoffee:      true,
	ProductCappuccino:        true,
	ProductCoffeeWithMilk:    true,
	ProductEspresso:          true,
	ProductEspressoChocolate: true,
	ProductKakao:             true,
	ProductHotWater:          true,
}

// Job represents a coffee machine job
type Job struct {
	JobID        string     `json:"jobId"`
	Product      Product    `json:"product"`
	JobStarted   time.Time  `json:"jobStarted"`
	JobReady     time.Time  `json:"jobReady"`
	JobRetrieved *time.Time `json:"jobRetrieved,omitempty"`
}

// StartJobRequest represents a request to start a job
type StartJobRequest struct {
	JobID   *string `json:"jobId,omitempty"`
	Product Product `json:"product"`
}

// Machine represents the coffee machine state
type Machine struct {
	mu         sync.RWMutex
	state      int
	jobs       map[string]*Job
	currentJob *Job
	readyJob   *Job
}

// NewMachine creates a new coffee machine instance
func NewMachine() *Machine {
	return &Machine{
		state: StateAvailable,
		jobs:  make(map[string]*Job),
	}
}

// GetState returns the current machine state
func (m *Machine) GetState() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// StartJob starts a new job if the machine is available
func (m *Machine) StartJob(req StartJobRequest) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if product is valid
	if !ValidProducts[req.Product] {
		return nil, fmt.Errorf("unsupported product: %s", req.Product)
	}

	// Check if machine can accept a new job
	if m.state != StateAvailable {
		return nil, fmt.Errorf("machine is not available")
	}

	// Generate or use provided job ID
	jobID := generateUUID()
	if req.JobID != nil && *req.JobID != "" {
		jobID = *req.JobID
	}

	// Check if job ID already exists
	if _, exists := m.jobs[jobID]; exists {
		return nil, fmt.Errorf("job ID already exists: %s", jobID)
	}

	now := time.Now()
	// Calculate brewing time (20-55 seconds)
	brewingTime := time.Duration(MinBrewingTime+int(time.Now().UnixNano()%int64(MaxBrewingTime-MinBrewingTime+1))) * time.Second

	job := &Job{
		JobID:      jobID,
		Product:    req.Product,
		JobStarted: now,
		JobReady:   now.Add(brewingTime),
	}

	m.jobs[jobID] = job
	m.currentJob = job
	m.state = StateBrewing

	// Start a goroutine to handle brewing completion
	go m.completeBrewing(job, brewingTime)

	return job, nil
}

// completeBrewing completes the brewing process after the specified duration
func (m *Machine) completeBrewing(job *Job, duration time.Duration) {
	time.Sleep(duration)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update state to blocked if this is still the current job
	if m.currentJob != nil && m.currentJob.JobID == job.JobID {
		m.currentJob = nil
		m.readyJob = job
		m.state = StateBlocked
	}
}

// RetrieveJob retrieves a ready job
func (m *Machine) RetrieveJob(jobID string) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	if job.JobRetrieved != nil {
		return nil, fmt.Errorf("job already retrieved")
	}

	now := time.Now()
	if now.Before(job.JobReady) {
		return nil, fmt.Errorf("job not ready yet")
	}

	job.JobRetrieved = &now

	// Update machine state
	if m.readyJob != nil && m.readyJob.JobID == jobID {
		m.readyJob = nil
		m.state = StateAvailable
	}

	return job, nil
}

// GetJob retrieves a job by ID without marking it as retrieved
func (m *Machine) GetJob(jobID string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.jobs[jobID]
	return job, exists
}

// GetAllJobs returns all jobs
func (m *Machine) GetAllJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// IsReady returns true if the machine is ready to accept new jobs
func (m *Machine) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == StateAvailable
}

// generateUUID generates a UUID v4-like identifier using crypto/rand
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(machine.jobs))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

var machine = NewMachine()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/start-job", handleStartJob)
	http.HandleFunc("/retrieve-job", handleRetrieveJob)
	http.HandleFunc("/healthz", handleHealthz)
	http.HandleFunc("/readyz", handleReadyz)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/history", handleHistory)

	log.Printf("Starting coffee machine server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleStartJob handles POST /start-job
// @Summary Start a new coffee job
// @Description Starts a new coffee brewing job. Returns 503 if machine is not available.
// @Tags jobs
// @Accept json
// @Produce json
// @Param request body StartJobRequest true "Job request"
// @Success 200 {object} Job
// @Failure 400 {string} string "Bad request"
// @Failure 503 {string} string "Service unavailable"
// @Router /start-job [post]
func handleStartJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	job, err := machine.StartJob(req)
	if err != nil {
		if err.Error() == "machine is not available" {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// handleRetrieveJob handles GET /retrieve-job?jobID=...
// @Summary Retrieve a completed job
// @Description Retrieves a ready coffee job by ID. Returns 503 if not ready, 404 if not found, 410 if already retrieved.
// @Tags jobs
// @Produce json
// @Param jobID query string true "Job ID"
// @Success 200 {object} Job
// @Failure 404 {string} string "Job not found"
// @Failure 410 {string} string "Job already retrieved"
// @Failure 503 {string} string "Service unavailable - job not ready"
// @Router /retrieve-job [get]
func handleRetrieveJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("jobID")
	if jobID == "" {
		http.Error(w, "jobID parameter is required", http.StatusBadRequest)
		return
	}

	job, err := machine.RetrieveJob(jobID)
	if err != nil {
		switch err.Error() {
		case "job not found":
			http.Error(w, err.Error(), http.StatusNotFound)
		case "job already retrieved":
			http.Error(w, err.Error(), http.StatusGone)
		case "job not ready yet":
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// handleHealthz handles GET /healthz
// @Summary Health check endpoint
// @Description Returns 200 if the service is healthy
// @Tags health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReadyz handles GET /readyz
// @Summary Readiness check endpoint
// @Description Returns 200 if the machine is ready to accept orders, 503 otherwise
// @Tags health
// @Produce plain
// @Success 200 {string} string "Ready"
// @Failure 503 {string} string "Not ready"
// @Router /readyz [get]
func handleReadyz(w http.ResponseWriter, r *http.Request) {
	if machine.IsReady() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not ready"))
	}
}

// handleStatus handles GET /status
// @Summary Machine status endpoint
// @Description Returns 200 when able to accept jobs, 503 when busy
// @Tags status
// @Produce plain
// @Success 200 {string} string "Available"
// @Failure 503 {string} string "Busy"
// @Router /status [get]
func handleStatus(w http.ResponseWriter, r *http.Request) {
	if machine.IsReady() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Available"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Busy"))
	}
}

// handleMetrics handles GET /metrics
// @Summary Prometheus metrics endpoint
// @Description Returns Prometheus metrics including coffee_machine_status gauge (0=available, 1=brewing, 2=blocked)
// @Tags metrics
// @Produce plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	state := machine.GetState()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP coffee_machine_status Current state of the coffee machine (0=available, 1=brewing, 2=blocked)\n")
	fmt.Fprintf(w, "# TYPE coffee_machine_status gauge\n")
	fmt.Fprintf(w, "coffee_machine_status %d\n", state)
}

// handleHistory handles GET /history
// @Summary Get job history
// @Description Returns the entire list of all jobs
// @Tags jobs
// @Produce json
// @Success 200 {array} Job
// @Router /history [get]
func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobs := machine.GetAllJobs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
