
# Drycc Storage v3

[![Build Status](https://drone.drycc.cc/api/badges/drycc/storage/status.svg)](https://drone.drycc.cc/drycc/storage)

Drycc (pronounced DAY-iss) Workflow is an open source Platform as a Service (PaaS) that adds a developer-friendly layer to any [Kubernetes](http://kubernetes.io) cluster, making it easy to deploy and manage applications on your own servers.

For more information about the Drycc workflow, please visit the main project page at https://github.com/drycc/workflow.

We welcome your input! If you have feedback, please submit an [issue][issues]. If you'd like to participate in development, please read the "Development" section below and submit a [pull request][prs].

# About

The Drycc storage component provides an [S3 API][s3-api] compatible object storage server, based on [Seaweedfs](https://github.com/chrislusf/seaweedfs), that can be run on Kubernetes. It's intended for use within the [Drycc v2 platform][drycc-docs] as an object storage server, but it's flexible enough to be run as a standalone pod on any Kubernetes cluster.

Note that in the default [Helm chart for the Drycc platform](https://github.com/drycc/charts/tree/main/drycc-dev), this component is used as a storage location for the following components:

- [drycc/postgres](https://github.com/drycc/postgres)
- [drycc/registry](https://github.com/drycc/registry)
- [drycc/builder](https://github.com/drycc/builder)

Also note that we aren't currently providing this component with any kind of persistent storage, but it may work with [persistent volumes](http://kubernetes.io/docs/user-guide/volumes/).

# Development

The Drycc project welcomes contributions from all developers. The high level process for development matches many other open source projects. See below for an outline.

* Fork this repository
* Make your changes
* Submit a [pull request][prs] (PR) to this repository with your changes, and unit tests whenever possible.
* If your PR fixes any [issues][issues], make sure you write Fixes #1234 in your PR description (where #1234 is the number of the issue you're closing)
* The Drycc core contributors will review your code. After each of them sign off on your code, they'll label your PR with `LGTM1` and `LGTM2` (respectively). Once that happens, you may merge.

## Dogfooding

Please follow the instructions on the [official Drycc docs][drycc-docs] to install and configure your Drycc cluster and all related tools, and deploy and configure an app on Drycc.


[install-k8s]: http://kubernetes.io/gettingstarted/
[s3-api]: http://docs.aws.amazon.com/AmazonS3/latest/API/APIRest.html
[issues]: https://github.com/drycc/storage/issues
[prs]: https://github.com/drycc/storage/pulls
[drycc-docs]: https://drycc.cc
[v2.18]: https://github.com/drycc/workflow/releases/tag/v2.18.0
