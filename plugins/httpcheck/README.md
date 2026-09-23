# httpcheck plugin

| Metadata        |           |
| ------------- |-----------|
|[Stability](/README.md#stability-levels)     | In Development   |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/pipe-cd/community-plugins?query=is%3Aissue%20is%3Aopen%20label%3Aplugin%2Fhttpcheck%20&label=open&color=orange)](https://github.com/pipe-cd/community-plugins/issues?q=is%3Aopen+is%3Aissue+label%3Aplugin%2Fhttpcheck) |
| [Code Owners](/CONTRIBUTING.md#becoming-a-code-owner)   |  [@Ahmad-Faraj](https://github.com/Ahmad-Faraj)  |

## Supported Features

- PipelineSync

## Overview

This is a stage plugin that checks an HTTP endpoint during a pipeline.

The stage polls a URL until it answers with the expected status code, or until a
timeout passes. It is meant as a gate between two other stages, for example to
confirm a new version actually responds before promoting it. Without it the usual
way to wait for a service to come up is a `WAIT` stage with a guessed duration.

There is no deploy target and no plugin scope configuration, so it can be added
to a pipeline of any application kind.

## Stages

### HTTP_CHECK stage

Sends a GET request to `url`, once immediately and then every `interval`, until
the response status equals `expectedCode`. If `timeout` passes first the stage
fails, which triggers rollback when the pipeline is configured for it.

The stage records its start time in stage metadata on the first run. If piped
restarts mid-stage, the remaining time is measured from that recorded start
rather than from the restart, so a restart does not reset the timeout. If the
budget has already passed when the stage resumes, one more probe still runs, so
an endpoint that is healthy by then can still pass.

Cancelling the deployment stops the polling. The plugin returns a failure status
and piped reports the stage as cancelled by the user.

## Plugin Configuration

### Plugin scope config

None. This plugin takes no piped scope configuration.

```yaml
kind: Piped
spec:
  plugins:
    - name: httpcheck
      port: 7002
      url: ...
```

### Deploy Target config

None. This plugin does not use deploy targets.

## Application Configuration

### Application scope options

None.

### Stage options

```yaml
kind: Application
spec:
  pipeline:
    stages:
      - name: K8S_CANARY_ROLLOUT
      - name: HTTP_CHECK
        with:
          url: https://example.com/healthz
          expectedCode: 200
          timeout: 2m
          interval: 5s
      - name: K8S_PRIMARY_ROLLOUT
```

#### HTTP_CHECK stage

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| url | string | The endpoint to check. Must be an http or https URL with a host. | Yes | |
| expectedCode | int | The HTTP status code treated as healthy. Must be between 100 and 599. | No | 200 |
| timeout | duration | How long to keep checking before failing the stage. | No | 1m |
| interval | duration | How long to wait between checks. Must be less than `timeout`. | No | 5s |
