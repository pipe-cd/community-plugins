# ansible plugin

| Metadata        |           |
| ------------- |-----------|
|[Stability](/README.md#stability-levels)     | In Development   |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/pipe-cd/community-plugins?query=is%3Aissue%20is%3Aopen%20label%3Aplugin%2Fansible%20&label=open&color=orange)](https://github.com/pipe-cd/community-plugins/issues?q=is%3Aopen+is%3Aissue+label%3Aplugin%2Fansible) |
| [Code Owners](/CONTRIBUTING.md#becoming-a-code-owner)   |  [@ntheanh201](https://github.com/@ntheanh201)  |

## Supported Features
<!-- 
- QuickSync
- PipelineSync
- Prune
- LiveState View
- DriftDetection
- PlanPreview
-->

<!-- You can add additional rows like 'PipelineSync by Istio', 'Analysis by <some-o11y-provider>', etc. -->

<!-- For a stages plugin, only PipelineSync would be supported in most cases. -->

## Overview

<!-- e.g. This is a plugin for deploying xxx. -->

## Stages

<!-- ### XXX stage -->
<!-- e.g. This stage shows a message on UI. -->

<!-- ### YYY stage -->

## Plugin Configuration

### Plugin scope config

<!-- 
Plugin scope config means 'HERE':

```yaml
kind: Piped
spec:
    plugins:
      - name: xxx
        port: 7002
        url: https://...
        config: # <-------- HERE
            aaa: ...
            bbb: ...
        deployTargets: 
          - name: cluster1
            ...
``` 

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| aaa | string | ... | Yes |  |
| bbb | map[string]string | ... | No |  |

-->

### Deploy Target config

<!-- 
Deploy Target config means 'HERE':

```yaml
kind: Piped
spec:
    plugins:
      - name: xxx
        port: 7002
        config:
            ...
        deployTargets: 
          - name: cluster1
            config:  # <-------- HERE
                ppp: ...
                qqq: ...
          - name: cluster2
            config: ...
``` 

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| ppp | string | ... | Yes | |
| qqq | map[string]string | ... | No | |

-->


## Application Configuration

### Application scope options
<!-- 
'Application scope options' means 'HERE':

```yaml
kind: Application
spec: 
    plugins: 
        xxx: 
          - name: xxx
            with: # <-------- HERE
                name:  ...
                labels: ...
                some: ...
```
-->

### Stage options
<!-- 

'Stage options' means 'HERE': 
```yaml
kind: Application
spec: 
    pipeline: 
        stages: 
          - name: xxx
            with: # <-------- HERE
                name:  ...
                labels: ...
                some: ...
```

#### XXX stage

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| name | string | The name to be shown in the stage. | Yes |
| labels | map[string]string | ... | No | | 
| some | [yourtype](#yourtype) | ... | No | |

##### yourtype

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| aaa | bool | ... | No | false |
| bbb | int | ... | No | 0 | 

#### YYY stage

| Field | Type | Description | Required | Default |
|-|-|-|-|-|
| messages | []string | The messages to be shown in the stage. | No | [""] |

-->

<!-- You can add additional sections if needed. -->