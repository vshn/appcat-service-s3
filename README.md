# provider-cloudscale

[![Build](https://img.shields.io/github/workflow/status/vshn/provider-cloudscale/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/provider-cloudscale)
[![Version](https://img.shields.io/github/v/release/vshn/provider-cloudscale)][releases]
[![Maintainability](https://img.shields.io/codeclimate/maintainability/vshn/provider-cloudscale)][codeclimate]
[![Coverage](https://img.shields.io/codeclimate/coverage/vshn/provider-cloudscale)][codeclimate]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/provider-cloudscale/total)][releases]

[build]: https://github.com/vshn/provider-cloudscale/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/provider-cloudscale/releases
[codeclimate]: https://codeclimate.com/github/vshn/provider-cloudscale

VSHN opinionated operator to deploy S3 resources on supported cloud providers.

https://vshn.github.io/provider-cloudscale/

## Local Development

### Requirements

* `docker`
* `go`
* `helm`
* `kubectl`
* [`kuttl`](https://kuttl.dev/)
* `yq`
* `sed` (or `gsed` for Mac)

Some other requirements (e.g. `kind`) will be compiled on-the-fly and put in the local cache dir `.kind` as needed.

### Common make targets

* `make build` to build the binary and docker image
* `make generate` to (re)generate additional code artifacts
* `make test` run test suite
* `make local-install` to install the operator in local cluster
* `make install-samples` to run the provider in local cluster and apply a sample instance
* `make run-operator` to run the code in operator mode against your current kubecontext

See all targets with `make help`

### QuickStart Demonstration

1. Get an API token cloudscale.ch
2. `export CLOUDSCALE_API_TOKEN=<the-token>`
3. `make local-install install-samples`

## Run Tests for XRD and Composition

Testing of the composition is handled by kuttl. You need it installed on your machine in order to run the tests.

Once you've installed it, you can simply run:
`make test-crossplane`

The tests themselves are located in the `test` folder.
