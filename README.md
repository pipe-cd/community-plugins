# community-plugins

This community-plugins repository is a collection of [PipeCD plugins](https://pipecd.dev/blog/2024/11/28/overview-of-the-plan-for-pluginnable-pipecd/) developed by the community.

_Note: Since the plugin system is still in development, the rules of this repository will be updated in the future._

## Objectives of This Repository

- **Accelerate**: To extend PipeCD by distributed developers without touching the core code.
- **Reliable**: To provide a more sustainable and reliable location compared to individual repos.
- **Unified**: To gather developers and users into an unified place.


## Community

See ['Community and Development' of pipe-cd/pipecd](https://github.com/pipe-cd/pipecd?tab=readme-ov-file#community-and-development).

## Usage

### Usage of piped and plugins

See https://github.com/pipe-cd/pipecd/blob/master/cmd/pipedv1/README-usage-alpha.md for now.

### Details of each plugin

Detailed usages of each plugin are available in each `plugins/<plugin-name>/README.md`.

## Stability Levels

_This will be changed in the future since we are still defining the process._

Note: Community plugins cannot be `Alpha` until we prepare the process, and cannot be `Beta` or `Stable` until the pipedv1 reaches beta.

1. **In Development**
   - Development is started.
   - Breaking changes are allowed.
2. **Alpha**
   - The plugin is ready for non-critical workloads.
   - Breaking changes are allowed in minor or patch releases. Notice will be provided in the release notes.｀
3. **Beta**
   - The plugin is ready for non-critical production workloads.
   - Breaking changes are allowed in minor releases with prior notice and efforts to minimize them, but NOT allowed in patch releases unless under special circumstances.
4. **Stable**
   - The plugin is ready for production use including critical workloads.
   - Breaking changes are NOT allowed in minor or patch releases, unless under special circumstances.

_**Unmaintained** and **Deprecated** levels: TBD_


### Criteria of Moving the Levels

_This will be changed in the future since we are still defining the process._

#### 1. In Development -> Alpha

Code Owners of a plugin can decide to move the plugin to Alpha whenever they want as long as all of the following criteria are met:

- At least one active code owner exists.
- README.md explains usage and the configurations.
- Have released the binaries of the plugin (at least `alpha` version)

To update to alpha:
1. A Code Owner updates the stability level in the README.md of the plugin.
2. Create a PR and get it merged.

#### 2. Alpha -> Beta

TBD

At least two active code owners are needed. There might be a submission form on Issues.

#### 3. Beta -> Stable

TBD

## Compatibility

TBD

