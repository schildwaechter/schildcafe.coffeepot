# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-15

### Added

- Initial implementation of coffee machine REST API
- Support for 8 product types: COFFEE, STRONG_COFFEE, CAPPUCCINO, COFFEE_WITH_MILK, ESPRESSO, ESPRESSO_CHOCOLATE, KAKAO, HOT_WATER
- POST `/start-job` endpoint to start new coffee brewing jobs
- GET `/retrieve-job` endpoint to retrieve completed jobs
- GET `/healthz` endpoint for health checks
- GET `/readyz` endpoint for Kubernetes readiness probes
- GET `/status` endpoint to check machine availability
- GET `/metrics` endpoint with Prometheus metrics (coffee_machine_status gauge)
- GET `/history` endpoint to retrieve all job history
- Machine state management (Available, Brewing, Blocked)
- Random brewing time between 20-55 seconds per product
- Job tracking with timestamps (started, ready, retrieved)
- UUID generation for job IDs using standard library
- Comprehensive unit tests for all functionality
- Dockerfile for containerized deployment
- OpenAPI 3.0 specification (openapi.yaml)
- .gitignore file for Go projects

### Technical Details

- Built with Go standard library only (no external dependencies)
- Thread-safe implementation using sync.RWMutex
- In-memory job storage
- Proper HTTP status codes (200, 400, 404, 410, 503)
- JSON request/response format
