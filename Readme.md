# SchildCafe Coffee Machine

This repository contains different branches where AI coding tools have been prompted to impement the Coffee Machine as described in the [Design.md](Design.md).

## Prompt

The following prompt was given:

> Build a simple Go application, using only the standard library.
> Implement a REST API for a coffee machine as outlined in the Design.md.
> The application should be packaged with a Dockerfile to run as a container in a Kubernetes cluster.
> Provide OpenAPI specs.

## Running locally

```bash
PORT=8080 go run .
```

## API

- Submit a job: `POST /start-job` with `{"product": "ESPRESSO"}` (optional `jobId`)
- Retrieve a job: `GET /retrieve-job?jobID=<id>`
- Health probes: `GET /healthz`, `GET /readyz`
- Status: `GET /status`
- Metrics: `GET /metrics` exposes `coffee_machine_status`
- History: `GET /history`
- OpenAPI: `GET /openapi.yaml`

## Docker

Build and run:

```bash
docker build -t coffee-machine .
docker run -p 8080:8080 coffee-machine
```
