# Contributing to Pulumi

First, thanks for contributing to Pulumi and helping make it better. We appreciate the help!
This repository is one of many across the Pulumi ecosystem and we welcome contributions to them all.

## Code of Conduct

Please make sure to read and observe our [Contributor Code of Conduct](https://github.com/pulumi/pulumi/blob/master/CODE-OF-CONDUCT.md).

## Communications

You are welcome to join the [Pulumi Community Slack](https://slack.pulumi.com/) for questions and a community of like-minded folks.
We discuss features and file bugs on GitHub via [Issues](https://github.com/pulumi/pulumi/issues) as well as [Discussions](https://github.com/pulumi/pulumi/discussions).
You can read about our [public roadmap](https://github.com/orgs/pulumi/projects/44) on the [Pulumi blog](https://www.pulumi.com/blog/relaunching-pulumis-public-roadmap/).

### Issues

Feel free to pick up any existing issue that looks interesting to you or fix a bug you stumble across while using Pulumi. No matter the size, we welcome all improvements.

### Feature Work

For larger features, we'd appreciate it if you open a [new issue](https://github.com/pulumi/pulumi/issues/new) before investing a lot of time so we can discuss the feature together.
Please also be sure to browse [current issues](https://github.com/pulumi/pulumi/issues) to make sure your issue is unique, to lighten the triage burden on our maintainers.
Finally, please limit your pull requests to contain only one feature at a time. Separating feature work into individual pull requests helps speed up code review and reduces the barrier to merge.

## Developing

### Setting up your Pulumi development environment

You'll want to install the following on your machine:

- [Go](https://go.dev/dl/) (a [supported version](https://go.dev/doc/devel/release#policy))
- [Golangci-lint](https://github.com/golangci/golangci-lint)
- [gofumpt](https://github.com/mvdan/gofumpt):
  see [installation](https://github.com/mvdan/gofumpt#installation) for editor setup instructions
- [Pulumictl](https://github.com/pulumi/pulumictl)

### Installing Pulumi dependencies on macOS

You can get all required dependencies with brew and npm

```bash
brew install go@1.21 golangci/tap/golangci-lint gofumpt pulumi/tap/pulumictl
```

### Make build system

We use `make` as our build system, so you'll want to install that as well, if you don't have it already.

### Building

`pulumi-converter-terraform` uses [Go modules](https://github.com/golang/go/wiki/Modules) to manage
dependencies. If you want to develop `pulumi-converter-terraform` itself, you'll need to have
[Go](https://golang.org/) installed in order to build. Once this prerequisite is installed, run the following
to build the `pulumi-converter-terraform` binary and install it into `$GOPATH/bin`:

```console
$ make install
```

Go should automatically handle pulling the dependencies for you.

If `$GOPATH/bin` is not on your path, you may want to move the `pulumi-converter-terraform` binary from `$GOPATH/bin`
into a directory that is on your path.

## Submitting a Pull Request

For contributors we use the [standard fork based workflow](https://gist.github.com/Chaser324/ce0505fbed06b947d962): Fork this repository, create a topic branch, and when ready, open a pull request from your fork.

Before you open a pull request, make sure all lint checks pass:

```bash
$ make lint
```

If you see formatting failures, fix them by running [gofumpt](https://github.com/mvdan/gofumpt) on your code:

```bash
$ gofumpt -w path/to/file.go
# or
$ gofumpt -w path/to/dir
```

## Getting Help

We're sure there are rough edges and we appreciate you helping out. If you want to talk with other folks in the Pulumi community (including members of the Pulumi team) come hang out in the `#contribute` channel on the [Pulumi Community Slack](https://slack.pulumi.com/).
