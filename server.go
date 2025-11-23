package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type server struct {
	machine *Machine
	logger  *log.Logger
}

func newServer(machine *Machine, logger *log.Logger) http.Handler {
	s := &server{
		machine: machine,
		logger:  logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/start-job", s.handleStartJob)
	mux.HandleFunc("/retrieve-job", s.handleRetrieveJob)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/openapi.yaml", s.handleOpenAPI)

	return loggingMiddleware(logger, mux)
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.machine.Ready() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
		return
	}

	http.Error(w, "busy", http.StatusServiceUnavailable)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	state := s.machine.Status()
	payload := map[string]any{
		"state": stateString(state),
		"code":  int(state),
	}
	status := http.StatusOK
	if state != StateAvailable {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, payload)
}

func (s *server) handleStartJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		JobID   string  `json:"jobId"`
		Product Product `json:"product"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	job, err := s.machine.StartJob(req.Product, req.JobID)
	if err != nil {
		switch {
		case errors.Is(err, ErrMachineBusy):
			http.Error(w, "machine unavailable", http.StatusServiceUnavailable)
		case errors.Is(err, ErrUnsupportedProduct):
			http.Error(w, "unsupported product", http.StatusBadRequest)
		case errors.Is(err, ErrJobIDExists):
			http.Error(w, "job ID already exists", http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (s *server) handleRetrieveJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("jobId")
	if jobID == "" {
		http.Error(w, "missing jobId", http.StatusBadRequest)
		return
	}

	job, err := s.machine.RetrieveJob(jobID)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			http.Error(w, "job not found", http.StatusNotFound)
		case errors.Is(err, ErrJobNotReady):
			http.Error(w, "job not ready", http.StatusServiceUnavailable)
		case errors.Is(err, ErrJobAlreadyRetrieved):
			http.Error(w, "job already retrieved", http.StatusGone)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.machine.History())
}

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	state := s.machine.Status()
	_, _ = w.Write([]byte("coffee_machine_status " + strconv.Itoa(int(state)) + "\n"))
}

func (s *server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("%s %s in %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func stateString(state MachineState) string {
	switch state {
	case StateAvailable:
		return "available"
	case StateBrewing:
		return "brewing"
	case StateBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

func start(ctx context.Context, port string, machine *Machine, logger *log.Logger) error {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: newServer(machine, logger),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	machine := NewMachine()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := start(ctx, port, machine, logger); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}
