# CONTRIBUTING.md

## Code of Conduct

PipeCD follows [the CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). Please read it to understand which actions are acceptable and which are not.

## Adding a New Plugin

Please follow the steps below. **You MUST NOT initialize a new plugin by opening a PR directly to avoid confusion.**

1. Please [open a 'New plugin proposal' issue](https://github.com/pipe-cd/community-plugins/issues/new?template=new-plugin.yaml).
2. A maintainer will create and merge a PR to initialize the plugin directory, CODEOWNERS, and so on by [the 'New Plugin' workflow](https://github.com/pipe-cd/community-plugins/actions/workflows/new-plugin.yaml).
3. The maintainer will close the issue, and you can start working on the plugin.

## Becoming a Code Owner

A code owner is responsible for maintaining the plugin, including triaging issues, reviewing PRs, and submitting bug fixes.

There are two ways to become a code owner:
- When [initializing a new plugin](#adding-a-new-plugin)
- When the current code owner(s) invite a new one. Then, follow these steps:
    1. Update the Code Owners in the README.md of the plugin.
    2. Execute `make sync/sync` to update the CODEOWNERS file.
    3. Create a PR and get it merged.

_What to do when a code owner becomes inactive?: TBD_

**NOTE: CodeOwners does NOT have write access to the repository yet. Please ask maintainers when editing Issues/PRs, including merging a PR. We're planning to add write access to CodeOwners in the future.**

## Issues

### Opening a new Issue

When opening a new issue, please make sure:
- to fill out the issue template for efficient communication.
- to search existing issues to avoid duplicates.

#### Security issues

**DO NOT open an Issue** for security problems. Instead, see [SECURITY.md in pipe-cd/pipecd](https://github.com/pipe-cd/pipecd/blob/master/SECURITY.md) and please send an email.

### Working on Issues

1. Before working on an issue, please leave a comment saying "I'd like to work on this." and we will assign the issue to you.
   - When you are assigned to an issue but seem inactive for some weeks, we will unassign you.
2. Before submitting a Pull Request, we expect you to investigate the issue and comment on what to do. Then you can discuss how to solve the issue and reduce the communication on the Pull Request.

## Pull Request

### Submitting a PR

When submitting a pull request, please ensure the following:

- **Issue assignment**: To avoid redundant work, make sure you are assigned to the issue.
- **Small PR**: Smaller PRs are much easier to review and are more likely to be merged.
- **DCO**: If you haven't signed off yet, see [License on contribution](#license-on-contribution).
- **`make precommit`**: To ensure your change will pass the CI.
- **Only main branch**: All PRs should be opened against the main branch.

### Reviewing

- Code Owners must lead reviewing for their plugin. `@pipe-cd/pipecd-approvers` is just a helper and does not assure the quality of the plugin.
- After CI passes and the code owner approves the PR, the code owner can merge it.

- _Note: When you are the only one code owner of a plugin and you want to submit a PR, `@pipe-cd/pipecd-approvers` can review it instantly. However, they are **not fully responsible** for the plugin and you should find a new code owner._

## Development, Testing

TBD

See [example-stage plugin](plugins/example-stage).

We're preparing a guide to use the SDK/API.


## Release Procedure

TBD

Each plugin will probably have its own version.

## License on contribution

For any code contribution, please carefully read the following documents:

- [LICENSE](/LICENSE)
- [Developer Certificate of Origin (DCO)](https://developercertificate.org/)

And signing off your commit with [`git commit -s`](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/managing-the-commit-signoff-policy-for-your-repository#about-commit-signoffs)
