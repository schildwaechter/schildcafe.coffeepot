# Coffee Machine

This is a coffee machine to produce different hot drinks running on a configurable port.
At least the following products must be supported.

* `COFFEE`
* `STRONG_COFFEE`
* `CAPPUCCINO`
* `COFFEE_WITH_MILK`
* `ESPRESSO`
* `ESPRESSO_CHOCOLATE`
* `KAKAO`
* `HOT_WATER`

The machine takes between 20 and 55 seconds for each product.

## Machine state

The machine can be in one of three states

1. Available to receive a new job
2. Brewing a job
3. Blocked by a ready job that has not been retrieved

All information is kept in memory.

## Main endpoints

The machine receives jobs as `json` posted to the `/start-job` endpoint.

This endpoint returns `503` if no job can be accepted.
Submitting unsupported products also returns an error.

The machine returns the (ready) coffee via get at the `/retrieve-job` endpoint
with the `jobID` as URL parameter.
If the job is not ready, the response is `503`,
if the job is not known the response is `404`
and if the job has been retrieved previously, the answer is `410`.

The machine stores each job in memory as an object

```json
{
  "jobId": UUID,
  "product": Product,
  "jobStarted": Timestamp,
  "jobReady": Timestamp,
  "jobRetrieved": Timestamp
}
```

The `jobId` can be sent with a job, or it gets set by the machine.
The `jobReady` is calculated at job submission, `jobRetrieved` defaults to Null until retrieved.

## Additional endpoints

* The machine has a `/healthz` health check endpoint
* The machine has a `/readyz` endpoint signaling whether it is ready to take an order.
* The machine has a `/status` endpoint that returns `200` when able to accept jobs, `503` when busy.
* The machine has a `/metrics` endpoint providing Prometheus metrics,
  specifically a state gauge `coffee_machine_status` with values 0, 1 or 2
* The machine has a `/history` endpoint where the entire list of all jobs can be retrieved

