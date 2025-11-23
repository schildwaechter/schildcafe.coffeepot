package main

import (
	"testing"
	"time"
)

func TestMachineHappyPath(t *testing.T) {
	m := NewMachine()
	m.brewTimeFn = func() time.Duration { return 10 * time.Millisecond }

	job, err := m.StartJob(ProductCoffee, "")
	if err != nil {
		t.Fatalf("start job: %v", err)
	}

	if m.Status() != StateBrewing {
		t.Fatalf("expected brewing state, got %v", m.Status())
	}

	time.Sleep(15 * time.Millisecond)

	if m.Status() != StateBlocked {
		t.Fatalf("expected blocked state after brew, got %v", m.Status())
	}

	result, err := m.RetrieveJob(job.JobID)
	if err != nil {
		t.Fatalf("retrieve job: %v", err)
	}

	if result.JobRetrieved == nil {
		t.Fatalf("expected retrieval timestamp to be set")
	}

	if m.Status() != StateAvailable || !m.Ready() {
		t.Fatalf("expected machine to be available after retrieval")
	}

	history := m.History()
	if len(history) != 1 {
		t.Fatalf("expected history length 1, got %d", len(history))
	}
	if history[0].JobID != job.JobID {
		t.Fatalf("expected history to contain job %s, got %s", job.JobID, history[0].JobID)
	}
}

func TestUnsupportedProductRejected(t *testing.T) {
	m := NewMachine()
	if _, err := m.StartJob(Product("INVALID"), ""); err == nil {
		t.Fatalf("expected unsupported product to fail")
	}
}

func TestRetrieveBeforeReadyFails(t *testing.T) {
	m := NewMachine()
	m.brewTimeFn = func() time.Duration { return 50 * time.Millisecond }

	job, err := m.StartJob(ProductEspresso, "")
	if err != nil {
		t.Fatalf("start job: %v", err)
	}

	if _, err := m.RetrieveJob(job.JobID); err == nil {
		t.Fatalf("expected retrieve to fail while brewing")
	}

	time.Sleep(60 * time.Millisecond)

	if _, err := m.RetrieveJob(job.JobID); err != nil {
		t.Fatalf("retrieve after ready: %v", err)
	}
}

func TestDuplicateJobID(t *testing.T) {
	m := NewMachine()
	if _, err := m.StartJob(ProductKakao, "job-1"); err != nil {
		t.Fatalf("first job: %v", err)
	}

	if _, err := m.StartJob(ProductKakao, "job-1"); err == nil {
		t.Fatalf("expected duplicate job ID to fail")
	}
}
