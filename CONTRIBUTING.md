# Contributing to bindplane-otel-collector

## Pull Requests

### How to Submit Pull Requests

Everyone is welcome to contribute code to `bindplane-otel-collector` via GitHub pull requests (PRs).

To create a new PR, fork the project in GitHub and clone the upstream repo:

```sh
$ git clone https://github.com/observIQ/bindplane-otel-collector
```

This would put the project in the `bindplane-otel-collector` directory in current working directory.

Enter the newly created directory and add your fork as a new remote:

```sh
$ git remote add <YOUR_FORK> git@github.com:<YOUR_GITHUB_USERNAME>/bindplane-otel-collector
```

Check out a new branch, make modifications, run linters and tests, and push the branch to your fork:

```sh
$ git checkout -b <YOUR_BRANCH_NAME>
# edit files
$ make test
$ git add
$ git commit
$ git push --set-upstream <YOUR_FORK> <YOUR_BRANCH_NAME>
```

Open a pull request from your fork and feature branch to the main branch of the `bindplane-otel-collector` repo.

**Note**: If the PR is not ready for review, mark it as [`draft`](https://github.blog/2019-02-14-introducing-draft-pull-requests/).

#### Commands to run before submitting PR

Our CI runs the following checks on each PR. You can run the following local commands to ensure your code is ready for PR:

- Build (`make collector`)
- CI Checks (`make ci-checks`)

### How to Receive Feedback

If you're stuck, tag a maintainer and ask a question. We're here to help each other.

### How to Get PRs Merged

A PR is considered to be **ready to merge** when:

- It has received approval from at least two maintainers.
- CI passes.
- Major feedback is resolved.

Tag a maintainer to request a merge once the above is complete.
