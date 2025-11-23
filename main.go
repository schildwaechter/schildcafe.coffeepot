package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Product represents the type of drink
type Product string

const (
	ProductCoffee             Product = "COFFEE"
	ProductStrongCoffee       Product = "STRONG_COFFEE"
	ProductCappuccino         Product = "CAPPUCCINO"
	ProductCoffeeWithMilk     Product = "COFFEE_WITH_MILK"
	ProductEspresso           Product = "ESPRESSO"
	ProductEspressoChocolate  Product = "ESPRESSO_CHOCOLATE"
	ProductKakao              Product = "KAKAO"
	ProductHotWater           Product = "HOT_WATER"
)

var supportedProducts = map[Product]bool{
	ProductCoffee:            true,
	ProductStrongCoffee:      true,
	ProductCappuccino:        true,
	ProductCoffeeWithMilk:    true,
	ProductEspresso:          true,
	ProductEspressoChocolate: true,
	ProductKakao:             true,
	ProductHotWater:          true,
}

// Job represents a coffee brewing job
type Job struct {
	JobID        string    `json:"jobId"`
	Product      Product   `json:"product"`
	JobStarted   time.Time `json:"jobStarted"`
	JobReady     time.Time `json:"jobReady"`
	JobRetrieved time.Time `json:"jobRetrieved,omitempty"` // omitempty for null behavior
}

// MachineState represents the current state of the machine
type MachineState int

const (
	StateAvailable MachineState = 0
	StateBrewing   MachineState = 1
	StateBlocked   MachineState = 2
)

// Machine holds the state of the coffee machine
type Machine struct {
	mu          sync.Mutex
	state       MachineState
	currentJob  *Job
	jobs        map[string]*Job
	history     []*Job
}

func NewMachine() *Machine {
	return &Machine{
		state:   StateAvailable,
		jobs:    make(map[string]*Job),
		history: make([]*Job, 0),
	}
}

func (m *Machine) StartJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Product Product `json:"product"`
		JobID   string  `json:"jobId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !supportedProducts[req.Product] {
		http.Error(w, "Unsupported product", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateAvailable {
		http.Error(w, "Machine is busy", http.StatusServiceUnavailable)
		return
	}

	jobID := req.JobID
	if jobID == "" {
		jobID = uuid.New().String()
	}

	// Calculate brewing time (20-55 seconds)
	// For simplicity, let's pick a deterministic or random value. 
    // The requirement says "takes between 20 and 55 seconds".
    // I'll use a fixed duration for now to make it predictable, or simple random.
    // Let's use 30 seconds.
	brewDuration := 30 * time.Second
	now := time.Now()
    
	job := &Job{
		JobID:      jobID,
		Product:    req.Product,
		JobStarted: now,
		JobReady:   now.Add(brewDuration),
	}

	m.jobs[jobID] = job
	m.history = append(m.history, job)
	m.currentJob = job
	m.state = StateBrewing

	// Start brewing in background
	go func(j *Job) {
		time.Sleep(brewDuration)
		m.mu.Lock()
		defer m.mu.Unlock()
        // Only update state if this is still the current job (it should be)
		if m.currentJob == j {
			m.state = StateBlocked
		}
	}(job)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (m *Machine) RetrieveJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("jobID")
	if jobID == "" {
		http.Error(w, "Missing jobID", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if !job.JobRetrieved.IsZero() {
		http.Error(w, "Job already retrieved", http.StatusGone)
		return
	}

	if time.Now().Before(job.JobReady) {
		http.Error(w, "Job not ready", http.StatusServiceUnavailable)
		return
	}

	// Job is ready and not retrieved.
	job.JobRetrieved = time.Now()
	
    // If this was the blocking job, free the machine
	if m.currentJob == job {
		m.currentJob = nil
		m.state = StateAvailable
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (m *Machine) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (m *Machine) Readyz(w http.ResponseWriter, r *http.Request) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.state == StateAvailable {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Busy"))
    }
}

func (m *Machine) Status(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == StateAvailable {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (m *Machine) Metrics(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Simple Prometheus format
	fmt.Fprintf(w, "# HELP coffee_machine_status The current status of the coffee machine (0=Available, 1=Brewing, 2=Blocked)\n")
	fmt.Fprintf(w, "# TYPE coffee_machine_status gauge\n")
	fmt.Fprintf(w, "coffee_machine_status %d\n", m.state)
}

func (m *Machine) History(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.history)
}

func main() {
	m := NewMachine()

	http.HandleFunc("/start-job", m.StartJob)
	http.HandleFunc("/retrieve-job", m.RetrieveJob)
	http.HandleFunc("/healthz", m.Healthz)
	http.HandleFunc("/readyz", m.Readyz)
	http.HandleFunc("/status", m.Status)
	http.HandleFunc("/metrics", m.Metrics)
	http.HandleFunc("/history", m.History)

	log.Println("Starting coffee machine on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
