# azure-functions plugin

| Metadata                                              |                                                                                                                                                                                                                                                                                                 |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Stability](/README.md#stability-levels)              | In Development                                                                                                                                                                                                                                                                                  |
| Issues                                                | [![Open issues](https://img.shields.io/github/issues-search/pipe-cd/community-plugins?query=is%3Aissue%20is%3Aopen%20label%3Aplugin%2Fazure-functions%20&label=open&color=orange)](https://github.com/pipe-cd/community-plugins/issues?q=is%3Aopen+is%3Aissue+label%3Aplugin%2Fazure-functions) |
| [Code Owners](/CONTRIBUTING.md#becoming-a-code-owner) | [@hiep-tk](https://github.com/@hiep-tk)                                                                                                                                                                                                                                                         |

## Supported Features

- QuickSync
- PipelineSync
- LiveState View
- Sync state reporting (ARM template deployments only; see [Limitations](#limitations))

Not implemented: Prune, DriftDetection (as a standalone feature), PlanPreview.

## Overview

This plugin deploys [Azure Functions](https://learn.microsoft.com/en-us/azure/azure-functions/) applications through PipeCD. It can, optionally, provision/update the underlying Azure infrastructure from an ARM template and then deploy a function app package (a zip file hosted at a URL, typically an Azure Blob Storage blob) to a Function App or one of its deployment slots. It also supports swapping two deployment slots (e.g. a staging/production blue-green swap) and reports the live state and sync state of the Function App back to PipeCD.

The plugin registers itself with piped under the internal name `azure` (see [main.go](main.go)) and implements both a deployment plugin ([deployment/plugin.go](deployment/plugin.go)) and a livestate plugin ([livestate/plugin.go](livestate/plugin.go)).

## Architecture

- `main.go` registers the deployment and livestate plugins with the piped plugin SDK.
- `config/` defines the plugin's configuration types:
  - `AzureDeployTargetConfig` — the Deploy Target config (Azure subscription).
  - `AzureApplicationSpec` / `FunctionsSpec` / `AzureDeployTemplate` — the Application config (`spec.plugins.azure`).
  - `AzureFunctionSyncStageConfig` / `AzureFunctionSwapStageConfig` — the per-stage `with` config.
- `deployment/` implements the `sdk.DeploymentPlugin` interface:
  - `plugin.go` — `FetchDefinedStages`, `BuildPipelineSyncStages`, `BuildQuickSyncStages`, `ExecuteStage`, `DetermineVersions`, `DetermineStrategy`.
  - `pipeline.go` — builds the list of stages for QuickSync and PipelineSync, including automatically appending the rollback stage when auto-rollback is enabled.
  - `func_sync.go`, `func_swap.go`, `func_rollback.go` — the implementation of each stage (see [Stages](#stages)).
- `livestate/plugin.go` implements `sdk.LivestatePlugin.GetLivestate`, returning the health of the Function App and its slots, plus a best-effort sync status.
- `provider/sdkclient.go` wraps the Azure SDK for Go behind an `AzureClient` interface used by both the deployment and livestate code. It performs authentication, ARM template deployment (including What-If), Function App/slot lookups and swaps, and package deployment (via Kudu zip-deploy or `WEBSITE_RUN_FROM_PACKAGE`).

The `provider` package also constructs SDK clients for App Service plans, Container Apps, AKS managed clusters, and Container Instances, and defines container-app helper methods (`ContainerAppEnvGet`, `ContainerAppEnvCreateOrUpdate`, `ContainerAppCreateOrUpdate`). None of these are called by the azure-functions deployment or livestate logic; they are currently unused within this plugin.

Only one deploy target is supported per application. Every stage and the `GetLivestate` call fail immediately if the request contains anything other than exactly one deploy target.

## Stages

### AZURE_FUNCTION_SYNC

Deploys the function app defined in `functionManifest`:

1. If `armTemplate` is set, deploys that ARM template to the resource group (`DeployARMTemplate`).
2. Verifies the Function App (and, if a slot is configured, the slot) now exists; fails if it still can't be found.
3. Deploys the package at `packageUri` to the Function App/slot:
   - If the current plan is a Linux Consumption ("Dynamic" SKU) plan, deploys by setting `WEBSITE_RUN_FROM_PACKAGE`/`WEBSITE_MOUNT_ENABLED` app settings and syncing.
   - Otherwise, deploys via the Kudu `zipdeploy` REST API and polls until it completes.

### AZURE_FUNCTION_SWAP

Swaps two deployment slots of the Function App named in `functionManifest`. At least one of `slot1`/`slot2` must be set; if only one is given, it is swapped with the production slot. The plugin first checks that all referenced slots already exist — this stage does not create resources, so it should generally follow an `AZURE_FUNCTION_SYNC` stage that has already provisioned them.

### AZURE_FUNCTION_ROLLBACK

Redeploys the function app using the manifest and package from the previously running deployment source, following the same logic as `AZURE_FUNCTION_SYNC` (ARM template deploy if set, then package deployment). This stage is automatically appended to the pipeline (for both QuickSync and PipelineSync) when auto-rollback is enabled; it is not a true "undo" of infrastructure changes, it simply re-deploys the previous package/template. See the comment in [func_rollback.go](deployment/func_rollback.go).

## Plugin Configuration

### Plugin scope config

This plugin does not read any plugin-scope config (`spec.plugins[].config` in the Piped config) — `sdk.ConfigNone` is used throughout. There are no fields to set here.

### Deploy Target config

Deploy Target config means 'HERE':

```yaml
kind: Piped
spec:
  plugins:
    - name: azure
      port: 7002
      url: file:///path/to/pipecd-plugin-azure
      deployTargets:
        - name: my-subscription
          config: # <-------- HERE
            subscriptionID: "00000000-0000-0000-0000-000000000000"
```

| Field          | Type   | Description                                                                                                                                                 | Required | Default |
| -------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- |
| subscriptionID | string | Azure Subscription ID that the deploy target's Azure clients operate against. It is validated (via a `GET` on the subscription) when the client is created. | Yes      |         |

## Application Configuration

### Application scope options

'Application scope options' means 'HERE':

```yaml
kind: Application
spec:
  plugins:
    azure:
      kind: function # <-------- HERE
      functionManifest: # <-------- HERE
        functionName: ...
        resourceGroupName: ...
        packageUri: ...
```

| Field            | Type                            | Description                                                                                                                                                                  | Required | Default |
| ---------------- | ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- |
| kind             | string                          | Must be `function` (the only value currently supported by `config.FunctionKind`). Any other value causes `DetermineVersions` to panic and `GetLivestate` to return an error. | Yes      |         |
| functionManifest | [FunctionsSpec](#functionsspec) | The Function App to deploy.                                                                                                                                                  | Yes      |         |

##### FunctionsSpec

| Field             | Type                                        | Description                                                                                                                                                                                                         | Required | Default |
| ----------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- |
| functionName      | string                                      | Name of the Azure Function App (and, for `armTemplate`/`bicepTemplate`, of the deployment).                                                                                                                         | Yes      |         |
| resourceGroupName | string                                      | Resource group containing the Function App.                                                                                                                                                                         | Yes      |         |
| packageUri        | string                                      | URL of the deployment package (a zip file, typically in Azure Blob Storage). Also used as the `DetermineVersions` artifact version/name.                                                                            | Yes      |         |
| armTemplate       | [AzureDeployTemplate](#azuredeploytemplate) | ARM template to deploy before the package. When omitted, no infrastructure is provisioned, and the Function App/slots must already exist. Also required for sync-state reporting (see [Limitations](#limitations)). | No       |         |
| bicepTemplate     | [AzureDeployTemplate](#azuredeploytemplate) | Declared in the config struct, but **not read or used anywhere in the plugin's code**. Setting it currently has no effect.                                                                                          | No       |         |

##### AzureDeployTemplate

| Field                   | Type   | Description                                                                                                                                                  | Required | Default |
| ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- | ------- |
| deploymentName          | string | Name of the ARM deployment.                                                                                                                                  | Yes      |         |
| deploymentTemplateFile  | string | Path, relative to the application directory, of the ARM template JSON file.                                                                                  | Yes      |         |
| deploymentParameterFile | string | Path, relative to the application directory, of the ARM parameters JSON file. Its top-level `parameters` object must be a map of `{ "value": ... }` entries. | Yes      |         |

### Stage options

'Stage options' means 'HERE':

```yaml
kind: Application
spec:
  pipeline:
    stages:
      - name: AZURE_FUNCTION_SYNC
        with: # <-------- HERE
          slot: ...
```

#### AZURE_FUNCTION_SYNC stage

| Field | Type   | Description                                                    | Required | Default |
| ----- | ------ | -------------------------------------------------------------- | -------- | ------- |
| slot  | string | Deployment slot to deploy to. Empty means the production slot. | No       | ""      |

#### AZURE_FUNCTION_SWAP stage

| Field | Type   | Description                                           | Required | Default |
| ----- | ------ | ----------------------------------------------------- | -------- | ------- |
| slot1 | string | First slot to swap. Empty means the production slot.  | No\*     | ""      |
| slot2 | string | Second slot to swap. Empty means the production slot. | No\*     | ""      |

\* At least one of `slot1`/`slot2` must be non-empty, or the stage fails.

#### AZURE_FUNCTION_ROLLBACK stage

Automatically inserted for rollback; it has no `with` options of its own (it reuses the sync-stage config shape internally, but is not intended to be configured directly in a pipeline).

## Required Environment

- Go 1.24.3 (see [go.mod](go.mod)) to build the plugin.
- Network access from piped to the Azure Resource Manager and Azure Storage endpoints for the target subscription.
- No plugin-specific environment variables are read directly by the code; the credential chain described in [Authentication](#authentication) does read standard Azure SDK environment variables.

## Authentication

The plugin authenticates using [`azidentity.NewDefaultAzureCredential`](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication), i.e. Azure's default credential chain (environment variables for a service principal or workload identity, managed identity, the Azure CLI's logged-in account, etc., depending on what's available in piped's runtime environment). There is no plugin config field for supplying credentials directly.

On top of Azure AD authentication, the plugin needs Azure RBAC permissions sufficient to:

- Read the target subscription (`Microsoft.Resources/subscriptions/read`, used to validate `subscriptionID`).
- Read the resource group and create/update/what-if ARM deployments in it, when `armTemplate` is set.
- Read and update the Function App/Web App and its deployment slots (get, list slots, swap slots, list/update application settings, sync triggers, list publishing credentials).
- List storage account keys for the storage account hosting `packageUri`, when that URL does not already carry a SAS token (`StorageBlobSASSign` generates a 7×24×520-hour read-only SAS from the account key in that case).

For package deployment over Kudu (`FunctionKuduDeploymentX`), the plugin uses SCM basic-auth publishing credentials if basic auth is enabled on the app, and otherwise falls back to an Azure AD bearer token scoped to `https://management.core.windows.net/.default`.

## Examples

### Piped Deploy Target

```yaml
kind: Piped
spec:
  plugins:
    - name: azure
      port: 7002
      url: file:///path/to/pipecd-plugin-azure
      deployTargets:
        - name: my-subscription
          config:
            subscriptionID: "00000000-0000-0000-0000-000000000000"
```

### QuickSync application

```yaml
apiVersion: pipecd.dev/v1beta1
kind: Application
spec:
  name: my-function-app
  plugins:
    azure:
      kind: function
      functionManifest:
        functionName: my-function-app
        resourceGroupName: my-resource-group
        packageUri: https://mystorageaccount.blob.core.windows.net/artifacts/my-function.zip
        armTemplate:
          deploymentName: my-function-app-deployment
          deploymentTemplateFile: azuredeploy.json
          deploymentParameterFile: azuredeploy.parameters.json
```

Without an explicit `pipeline`, `DetermineStrategy` reports QuickSync, which runs `AZURE_FUNCTION_SYNC` against the production slot.

### Pipeline with a staging slot and swap

```yaml
apiVersion: pipecd.dev/v1beta1
kind: Application
spec:
  name: my-function-app
  plugins:
    azure:
      kind: function
      functionManifest:
        functionName: my-function-app
        resourceGroupName: my-resource-group
        packageUri: https://mystorageaccount.blob.core.windows.net/artifacts/my-function.zip
  pipeline:
    stages:
      - name: AZURE_FUNCTION_SYNC
        with:
          slot: staging
      - name: AZURE_FUNCTION_SWAP
        with:
          slot1: staging
          slot2: "" # production
```

Both `staging` and the production slot must already exist (e.g. created ahead of time, or via an `armTemplate` in an earlier deployment) since `AZURE_FUNCTION_SWAP` does not create slots.

## Troubleshooting

- **Stage fails with `Currently support only one deployment target, instead got N`**: the application's deploy targets list must resolve to exactly one entry for this plugin.
- **`Cannot find resource even after deploy template`**: after an (optional) ARM template deployment, the Function App or one of the requested slots still doesn't exist. Either add/fix `armTemplate` so it provisions everything referenced by `functionManifest`/stage `with` config, or pre-create the missing resources.
- **`Swap template requires at least one slot set` / `Sync stage should have resource existence guaranteed`**: `AZURE_FUNCTION_SWAP` requires `slot1` and/or `slot2`, and both referenced slots must already exist (typically from a prior `AZURE_FUNCTION_SYNC`).
- **Plugin process panics with `unimplemented stage: ...`**: `ExecuteStage` only handles `AZURE_FUNCTION_SYNC`, `AZURE_FUNCTION_SWAP`, and `AZURE_FUNCTION_ROLLBACK`. Any other stage name in the pipeline will panic the plugin.
- **Plugin process panics with `not supported kind: ...`**: `functionManifest`'s parent `kind` field must be `function`.
- **Live state shows `Sync state is not supported without template deployment`**: sync-state comparison is only computed when `armTemplate` is set; it uses ARM What-If against the ARM template, not against the deployed function package itself.
- **Zip deploy fails with a non-202 status, or times out**: check the Kudu `zipdeploy` response body logged in the stage error, and confirm SCM basic auth / Azure AD access to the target Function App from wherever piped runs.

## Development

- Module: `github.com/pipe-cd/community-plugins/plugins/azure-functions` (Go 1.24.3).
- Build: `make build/go PLUGINS=azure-functions` from the repository root (or `go build ./...` inside this directory).
- Lint: `make lint/go MODULES=./plugins/azure-functions` from the repository root.
- [Makefile](Makefile) includes the shared `Makefile.common` and does not currently override any targets for this plugin.
- Before submitting a PR, run `make precommit` from the repository root (lint, build, test, and sync of the plugins list/CODEOWNERS), and sign off commits per the [DCO requirement](/CONTRIBUTING.md#license-on-contribution).

## Testing

There are currently no automated tests (`*_test.go` files) in this plugin. `make precommit`/`test/go` will run successfully but exercise no plugin-specific test code.

## Limitations

- Only a single deploy target per application is supported.
- `functionManifest.kind` (application spec `kind`) must be `function`; any other value panics `DetermineVersions` and errors out `GetLivestate`.
- `ExecuteStage` panics on any stage name outside `AZURE_FUNCTION_SYNC`/`AZURE_FUNCTION_SWAP`/`AZURE_FUNCTION_ROLLBACK`.
- `bicepTemplate` is defined in the config but is never read by the plugin — Bicep template deployment is not implemented despite the field existing.
- `AZURE_FUNCTION_SWAP` never creates the slots it swaps; they must already exist.
- Rollback re-deploys the previous package/ARM template rather than performing a true infrastructure rollback (see the comment in [func_rollback.go](deployment/func_rollback.go)).
- Sync-state / drift reporting only works when `armTemplate` is configured, and it compares against the ARM template (via What-If), not against the deployed function package/code.
- No Prune or PlanPreview support.
- No plugin-scope config (`spec.plugins[].config`) is supported.
- SDK clients for App Service plans, Container Apps, AKS, and Container Instances are constructed in `provider/sdkclient.go` but unused by this plugin's deployment/livestate logic.
- No automated tests exist yet for this plugin.
- The plugin is at the **In Development** stability level; breaking changes are possible.
