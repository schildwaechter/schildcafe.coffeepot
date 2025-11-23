package main

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	"sync"
	"time"
)

// Product is one of the supported beverages the machine can brew.
type Product string

// Supported products.
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

var supportedProducts = map[Product]struct{}{
	ProductCoffee:            {},
	ProductStrongCoffee:      {},
	ProductCappuccino:        {},
	ProductCoffeeWithMilk:    {},
	ProductEspresso:          {},
	ProductEspressoChocolate: {},
	ProductKakao:             {},
	ProductHotWater:          {},
}

// MachineState tracks the lifecycle of a job on the machine.
type MachineState int

const (
	StateAvailable MachineState = iota
	StateBrewing
	StateBlocked
)

const (
	minBrewSeconds = 20
	maxBrewSeconds = 55
)

// Job holds all information for a single brew request.
type Job struct {
	JobID        string     `json:"jobId"`
	Product      Product    `json:"product"`
	JobStarted   time.Time  `json:"jobStarted"`
	JobReady     time.Time  `json:"jobReady"`
	JobRetrieved *time.Time `json:"jobRetrieved,omitempty"`
}

var (
	ErrMachineBusy         = errors.New("machine not available to accept jobs")
	ErrUnsupportedProduct  = errors.New("unsupported product")
	ErrJobNotFound         = errors.New("job not found")
	ErrJobNotReady         = errors.New("job not ready")
	ErrJobAlreadyRetrieved = errors.New("job already retrieved")
	ErrJobIDExists         = errors.New("job ID already exists")
)

// Machine manages a single in-memory coffee machine instance.
type Machine struct {
	mu         sync.Mutex
	state      MachineState
	jobs       map[string]*Job
	history    []string
	currentJob string
	rand       *mrand.Rand
	brewTimeFn func() time.Duration
}

// NewMachine constructs an idle machine ready to accept jobs.
func NewMachine() *Machine {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	return &Machine{
		state:      StateAvailable,
		jobs:       make(map[string]*Job),
		history:    make([]string, 0),
		rand:       r,
		brewTimeFn: defaultBrewTime(r),
	}
}

func defaultBrewTime(r *mrand.Rand) func() time.Duration {
	return func() time.Duration {
		seconds := r.Intn(maxBrewSeconds-minBrewSeconds+1) + minBrewSeconds
		return time.Duration(seconds) * time.Second
	}
}

// Ready reports whether the machine can accept a new job.
func (m *Machine) Ready() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state == StateAvailable
}

// Status returns the current machine state.
func (m *Machine) Status() MachineState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// StartJob queues a brew request if the machine is idle.
func (m *Machine) StartJob(product Product, jobID string) (Job, error) {
	if !product.valid() {
		return Job{}, ErrUnsupportedProduct
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateAvailable {
		return Job{}, ErrMachineBusy
	}

	if jobID == "" {
		var err error
		jobID, err = generateJobID()
		if err != nil {
			return Job{}, err
		}
	}

	if _, exists := m.jobs[jobID]; exists {
		return Job{}, ErrJobIDExists
	}

	started := time.Now()
	brewDuration := m.brewTimeFn()
	readyAt := started.Add(brewDuration)

	job := &Job{
		JobID:      jobID,
		Product:    product,
		JobStarted: started,
		JobReady:   readyAt,
	}

	m.jobs[jobID] = job
	m.history = append(m.history, jobID)
	m.currentJob = jobID
	m.state = StateBrewing

	go m.awaitCompletion(jobID, brewDuration)

	return *job, nil
}

// RetrieveJob returns a brewed job if it is ready and not yet retrieved.
func (m *Machine) RetrieveJob(jobID string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return Job{}, ErrJobNotFound
	}

	if job.JobRetrieved != nil {
		return Job{}, ErrJobAlreadyRetrieved
	}

	if time.Now().Before(job.JobReady) {
		return Job{}, ErrJobNotReady
	}

	now := time.Now()
	job.JobRetrieved = &now

	if m.currentJob == jobID {
		m.currentJob = ""
		m.state = StateAvailable
	}

	return *job, nil
}

// History returns a snapshot of all jobs in submission order.
func (m *Machine) History() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobs := make([]Job, 0, len(m.history))
	for _, id := range m.history {
		if job, ok := m.jobs[id]; ok {
			jobs = append(jobs, *job)
		}
	}

	return jobs
}

func (m *Machine) awaitCompletion(jobID string, duration time.Duration) {
	time.Sleep(duration)

	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok && job.JobRetrieved == nil {
		m.state = StateBlocked
	}
}

func (p Product) valid() bool {
	_, ok := supportedProducts[p]
	return ok
}

func generateJobID() (string, error) {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate job id: %w", err)
	}

	// Set variant and version bits for UUID4 compatibility.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	), nil
}
