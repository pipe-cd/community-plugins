# sqldef plugin

| Metadata                                              |                                                                                                                                                                                                                                                                               |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Stability](/README.md#stability-levels)              | In Development                                                                                                                                                                                                                                                                |
| Issues                                                | [![Open issues](https://img.shields.io/github/issues-search/pipe-cd/community-plugins?query=is%3Aissue%20is%3Aopen%20label%3Aplugin%2Fsqldef%20&label=open&color=orange)](https://github.com/pipe-cd/community-plugins/issues?q=is%3Aopen+is%3Aissue+label%3Aplugin%2Fsqldef) |
| [Code Owners](/CONTRIBUTING.md#becoming-a-code-owner) | [@kadai0308](https://github.com/@kadai0308)                                                                                                                                                                                                                                   |

## Supported Features

- QuickSync
- PipelineSync

Not implemented: Prune, LiveState View, DriftDetection, PlanPreview. `main.go` only registers a deployment plugin (`sdk.WithDeploymentPlugin`); no livestate or plan-preview plugin is registered.

## Overview

This plugin deploys database schema changes using [sqldef](https://github.com/sqldef/sqldef), a schema-as-code tool that diffs a desired-state SQL schema file against a live database and applies the difference (idempotent, declarative migrations — no versioned migration files). Only **MySQL** is currently supported, via the `mysqldef` binary; the config type also declares `psql`, `sqlite`, and `mssql` as possible database types, but the plugin rejects them at runtime (see [Limitations](#limitations)).

For each deploy target, the plugin downloads the appropriate `mysqldef` binary, locates a single `.sql` schema file inside the application's deployment source directory, and runs `mysqldef` against the target database, either as a dry run (`SQLDEF_PLAN`) or applying the changes (`SQLDEF_APPLY`/`SQLDEF_ROLLBACK`).

## Architecture

- `main.go` registers the deployment plugin with the piped plugin SDK (`sdk.NewPlugin("0.0.1", sdk.WithDeploymentPlugin(...))`). It does not pass an explicit plugin name to the SDK, so the plugin's identity as seen by piped comes entirely from the `name:` given to it in the Piped config's `plugins:` list.
- `config/config.go` defines the plugin's configuration types:
  - `Config` — the plugin-scope config. It is an empty struct; no fields are read.
  - `DeployTargetConfig` — the Deploy Target config: `db_type`, `username`, `password`, `host`, `port`, `db_name`.
  - `ApplicationConfigSpec` — the application-scope config. It is also an empty struct; no fields are read.
- `deployment/plugin.go` implements `sdk.DeploymentPlugin`:
  - `FetchDefinedStages` — returns `SQLDEF_PLAN`, `SQLDEF_APPLY`, `SQLDEF_ROLLBACK`.
  - `BuildPipelineSyncStages` — passes through the stages requested for the pipeline as-is, and appends a `SQLDEF_ROLLBACK` stage (at the minimum index of the requested stages) when rollback is requested.
  - `BuildQuickSyncStages` — always includes `SQLDEF_APPLY`, plus `SQLDEF_ROLLBACK` when rollback is requested.
  - `ExecuteStage` — dispatches to `executePlanStage`, `executeApplyStage`, or `executeRollbackStage` based on the stage name; panics with `"unimplemented stage"` for any other name.
  - `DetermineVersions` — reports the deployment's commit hash as the single artifact version.
  - `DetermineStrategy` — returns `(nil, nil)`, the SDK's documented convention for "this plugin has no strategy-specific logic"; the sync strategy is left to piped's common logic (e.g. pipeline length, first deployment).
- `deployment/stage_plan.go`, `stage_apply.go`, `stage_rollback.go` implement the three stages. Each one, for every deploy target, in order: downloads the `mysqldef` binary via the tool registry, finds a schema file via `findFirstSQLFile`, calls `provider.SqldefProvider.Init`, then `Execute`. They return `sdk.StageStatusFailure` immediately on the first error from any deploy target — remaining targets in the list are not attempted, and no automatic rollback of already-applied targets happens within the same stage run.
- `deployment/utils.go` — `findFirstSQLFile` walks the application directory and returns the path of the first file whose name ends in `.sql`.
- `provider/sqldef.go` — `SqldefProviderImpl` shells out to the `mysqldef` binary (`os/exec`) to run `Execute` (apply or `--dry-run`) and `ShowCurrentSchema` (`--export`). `ShowCurrentSchema` is defined on the `SqldefProvider` interface and implemented, but it is not called anywhere in `deployment/`.
- `toolregistry/` downloads and caches the `mysqldef` binary. `registry.go`'s `DownloadBinary` only handles `config.DBTypeMySQL`; every other `DBType` returns an `"unsupported database type"` error. `script.go` holds the install script (`curl` + `unzip` of a GitHub release asset) used via the SDK's tool registry `InstallTool`.

## Stages

### SQLDEF_PLAN

For each deploy target: downloads/installs `mysqldef`, finds the schema file, then runs `mysqldef ... --dry-run` and logs the SQL statements that would be executed, without changing the database.

### SQLDEF_APPLY

Same steps as `SQLDEF_PLAN`, but runs `mysqldef` without `--dry-run`, applying the schema file to the target database. `mysqldef` is always invoked with `--enable-drop` (not configurable), so tables/columns/indexes present in the database but absent from the schema file are dropped.

### SQLDEF_ROLLBACK

Same steps as `SQLDEF_APPLY` (applies, does not dry-run), except the schema file is looked up in the _previously running_ deployment source's application directory (`RunningDeploymentSource`) instead of the target one. There is no database snapshot/backup involved — rollback simply re-applies whatever schema file was part of the previous deployment. This stage is one of the plugin's `FetchDefinedStages`, so it can either be auto-appended by piped when auto-rollback is enabled, or referenced directly as a regular stage in a pipeline's `stages` list.

## Plugin Configuration

### Plugin scope config

This plugin does not define any plugin-scope config fields — `config.Config` is an empty struct. A `config:` block under `spec.plugins[].config` in the Piped configuration, if present, is not read.

### Deploy Target config

Deploy Target config means 'HERE':

```yaml
kind: Piped
spec:
  plugins:
    - name: sqldef
      port: 7002
      url: file:///path/to/pipecd-plugin-sqldef
      deployTargets:
        - name: my-mysql
          config: # <-------- HERE
            db_type: mysql
            username: root
            password: password
            host: 127.0.0.1
            port: "3306"
            db_name: my_database
```

| Field    | Type   | Description                                                                                                                                                                                                           | Required | Default |
| -------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- |
| db_type  | string | Database engine. `mysql` is the only value that currently works — `psql`, `sqlite`, and `mssql` are defined by the type but rejected at runtime with `unsupported database type: ..., currently only support: mysql`. | Yes      |         |
| username | string | Database username, passed to `mysqldef` via `-u`.                                                                                                                                                                     | Yes      |         |
| password | string | Database password, passed to `mysqldef` via `-p` as a literal CLI argument (see [Authentication](#authentication)).                                                                                                   | Yes      |         |
| host     | string | Database host, passed via `-h`.                                                                                                                                                                                       | Yes      |         |
| port     | string | Database port, passed via `-P`.                                                                                                                                                                                       | Yes      |         |
| db_name  | string | Name of the database to manage.                                                                                                                                                                                       | Yes      |         |

## Application Configuration

### Application scope options

This plugin does not define any application-scope config either — `config.ApplicationConfigSpec` is an empty struct, and the raw stage config bytes (`ExecuteStageInput.Request.StageConfig`) are never unmarshalled anywhere in `deployment/`. Connection details come entirely from the Deploy Target config, and the schema to apply is discovered from the application's source directory (see [Stages](#stages)), not from any application-level YAML.

### Stage options

None of `SQLDEF_PLAN`/`SQLDEF_APPLY`/`SQLDEF_ROLLBACK` read a `with:` block — any fields placed there currently have no effect on the plugin's behavior.

## Required Environment

- Go 1.24.3 (see [go.mod](go.mod)) to build the plugin.
- Network access from the piped host to `github.com/sqldef/sqldef` releases, to download the `mysqldef` binary the first time a given version is needed (cached afterward by the SDK's tool registry).
- `curl` and `unzip` available on the piped host's `PATH` — the install script in [toolregistry/script.go](toolregistry/script.go) shells out to both to fetch and unpack the release zip.
- Network access from the piped host to the target MySQL server (`host`/`port` in the Deploy Target config).
- No plugin-specific environment variables are read by the code.

## Authentication

The plugin does not integrate with any secret manager or credential provider. Database credentials (`username`/`password`) are plain string fields in the Deploy Target config, and are passed through to `mysqldef` as literal `-u`/`-p` command-line arguments on every stage execution — as a result, they may be visible to other processes on the piped host that can read that process's command line (e.g. via `ps`). There is no TLS/SSL option exposed in the plugin's config; any such behavior is whatever `mysqldef` does by default for the given host/port.

The configured MySQL user needs whatever privileges the schema file's changes require, including `DROP`, since `SQLDEF_APPLY`/`SQLDEF_ROLLBACK` always run with `--enable-drop`.

## Examples

### Piped Deploy Target

```yaml
kind: Piped
spec:
  plugins:
    - name: sqldef
      port: 7002
      url: file:///path/to/pipecd-plugin-sqldef
      deployTargets:
        - name: my-mysql
          config:
            db_type: mysql
            username: root
            password: password
            host: 127.0.0.1
            port: "3306"
            db_name: my_database
```

### QuickSync application

The application directory must contain at least one `.sql` file (e.g. `schema.sql`) describing the desired schema.

```yaml
apiVersion: pipecd.dev/v1beta1
kind: Application
spec:
  name: my-database
  plugins:
    sqldef: {}
```

Without an explicit `pipeline`, QuickSync runs `SQLDEF_APPLY` directly against the schema file found in the application directory.

### Pipeline with an explicit plan step

```yaml
apiVersion: pipecd.dev/v1beta1
kind: Application
spec:
  name: my-database
  plugins:
    sqldef: {}
  pipeline:
    stages:
      - name: SQLDEF_PLAN
      - name: SQLDEF_APPLY
```

`SQLDEF_PLAN` logs the pending diff as a dry run; `SQLDEF_APPLY` then applies it. Both stages independently re-run `mysqldef` against the live database, so the plan's output can go stale if the database changes between the two stages.

## Troubleshooting

- **Stage fails with `unsupported database type: <type>, currently only support: mysql`**: `db_type` in the Deploy Target config must be `mysql`; `psql`, `sqlite`, and `mssql` are declared in the config type but not implemented.
- **Stage fails while finding the schema file (e.g. file does not exist)**: `findFirstSQLFile` found no file ending in `.sql` anywhere under the application's deployment source directory. Make sure at least one `.sql` file is present.
- **Unexpected tables/columns/indexes get dropped by `SQLDEF_APPLY`**: `mysqldef` is always invoked with `--enable-drop`; anything present in the live database but missing from the schema file will be dropped. Make sure the schema file describes the full desired state, not just an incremental change.
- **Log line `Failed while plan the deployment (...)` / `Failed while applying the deployment (...)`**: these wrap the underlying error from downloading the binary, finding the schema file, or running `mysqldef` — check the accompanying error/stderr text logged with it.
- **Plugin process panics with `unimplemented stage`**: `ExecuteStage` only handles `SQLDEF_PLAN`, `SQLDEF_APPLY`, and `SQLDEF_ROLLBACK`; check that the pipeline's stage name matches one of these exactly.
- **A multi deploy-target stage stops partway through**: the stage loop returns failure on the first deploy target that errors and does not process the remaining targets, and does not undo changes already applied to earlier targets in that same run.

## Development

- Module: `github.com/pipe-cd/community-plugins/plugins/sqldef` (Go 1.24.3).
- Build: `make build/go PLUGINS=sqldef` from the repository root (or `go build ./...` inside this directory).
- Lint: `make lint/go MODULES=./plugins/sqldef` from the repository root.
- [Makefile](Makefile) includes the shared `Makefile.common` and does not currently override any targets for this plugin.
- Before submitting a PR, run `make precommit` from the repository root (lint, build, test, and sync of the plugins list/CODEOWNERS), and sign off commits per the [DCO requirement](/CONTRIBUTING.md#license-on-contribution).

## Testing

Unit tests exist for:

- `deployment/plugin_test.go` — `BuildPipelineSyncStages`, `BuildQuickSyncStages`.
- `deployment/stage_plan_test.go` and `deployment/stage_apply_test.go` — `executePlanStage`/`executeApplyStage`, covering the happy path, unsupported DB types, missing schema files, `mysqldef` execution errors, and multiple deploy targets (including partial failure and binary caching across targets).
- `toolregistry/registry_test.go` — `DownloadBinary` for each `DBType` value.

Tests mock `provider.SqldefProvider` with `testify/mock`, and use the SDK's `toolregistrytest`/`logpersistertest` packages to fake the tool registry and log persister, so no real database or `mysqldef` binary is needed to run the suite.

There are currently no tests for `deployment/stage_rollback.go` (`SQLDEF_ROLLBACK`), `deployment/utils.go` (`findFirstSQLFile`), or `provider/sqldef.go` (`SqldefProviderImpl`).

Run with `go test ./...` inside this directory, or `make test/go PLUGINS=sqldef` / `make precommit` from the repository root.

## Limitations

- Only `mysql` is a working `db_type`; `psql`, `sqlite`, and `mssql` are declared in the config type but always fail at runtime.
- The `mysqldef` version is hardcoded to a default (`2.0.4`) — there is no config field to select a different version.
- Only a single schema file per deploy target is used: `findFirstSQLFile` returns the first `.sql` file encountered while walking the application directory: there's no way to select a specific file or combine multiple `.sql` files.
- `SQLDEF_APPLY`/`SQLDEF_ROLLBACK` always pass `--enable-drop` to `mysqldef` with no way to disable it, so unmodeled tables/columns/indexes are dropped by design.
- Multi-target stage execution is not atomic: the first failing deploy target aborts the stage, remaining targets are skipped, and targets already applied earlier in the same run are not rolled back automatically.
- `SQLDEF_ROLLBACK` re-applies the schema file from the previously running deployment source; it does not use a database backup/snapshot, so it is only as correct as that file's match to the database's actual prior state.
- No plugin-scope or application-scope config is read (`config.Config` and `config.ApplicationConfigSpec` are both empty structs), and no per-stage `with:` options are read.
- Credentials are plain strings with no secret-reference mechanism in this plugin's own config, and are passed to `mysqldef` as literal CLI arguments (see [Authentication](#authentication)).
- No LiveState plugin, PlanPreview plugin, Prune, or standalone DriftDetection is implemented.
- `provider.SqldefProvider.ShowCurrentSchema` is implemented but never called by the plugin's deployment logic.
- `ExecuteStage` panics on any stage name outside `SQLDEF_PLAN`/`SQLDEF_APPLY`/`SQLDEF_ROLLBACK`.
- No automated tests exist yet for the rollback stage, `findFirstSQLFile`, or `SqldefProviderImpl`.
- The plugin is at the **In Development** stability level; breaking changes are possible.
