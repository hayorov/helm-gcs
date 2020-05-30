## WARNING: master switched to HELM 3
### for helm 2 use `install ... --version 0.2.2`
<p align="center">
	<img src="https://raw.githubusercontent.com/hayorov/helm-gcs/master/assets/helm-gcs-logo.png" alt="helm-gcs logo"/>
</p>

# helm-gcs [![Build Status](https://travis-ci.org/hayorov/helm-gcs.svg?branch=master)](https://travis-ci.org/hayorov/helm-gcs)

`helm-gcs` is a [helm](https://github.com/kubernetes/helm) plugin that allows you to manage private helm repositories on [Google Cloud Storage](https://cloud.google.com/storage/) aka buckets.

## Installation

Install the stable version:
```shell
$ helm plugin install https://github.com/hayorov/helm-gcs.git
```

Install a specific version:
```shell
$ helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.3.2
```

## Quick start

```shell
# Init a new repository
$ helm gcs init gs://bucket/path

# Add your repository to Helm
$ helm repo add repo-name gs://bucket/path

# Push a chart to your repository
$ helm gcs push chart.tar.gz repo-name

# Update Helm cache
$ helm repo update

# Fetch the chart
$ helm fetch repo-name/chart

# Remove the chart
$ helm gcs rm chart repo-name
```

## Documentation

### Authentification

To authenticate against GCS you can:

 -   Use the [application default credentials](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/)

 -   Use a service account via the global flag `--service-account`

See [the GCP documentation](https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application) for more information.


### Create a repository

First, you need to [create a bucket on GCS](https://cloud.google.com/storage/docs/creating-buckets), which will be used by the plugin to store your charts.

Then you have to initialize a repository at a specific location in your bucket:

```shell
$ helm gcs init gs://your-bucket/path
```

>   You can create a repository anywhere in your bucket.

>   This command does nothing if a repository already exists at the given location.

You can now add the repository to helm:
```shell
$ helm repo add my-repository gs://your-bucket/path
```

### Push a chart

Package the chart:
```shell
$ helm package my-chart
```
This will create a file `my-chart-<semver>.tgz`.

Now, to push the chart to the repository `my-repository`:

```shell
$ helm gcs push my-chart-<semver>.tgz my-repository
```

If you got this error:
```shell
Error: update index file: index is out-of-date
```

That means that someone/something updated the same repository, at the same time as you. You just need to execute the command again or, next time, use the `--retry` flag to automatically retry to push the chart.

Once the chart is uploaded, use helm to fetch it:

```shell
# Update local repo cache if necessary
# $ helm repo update

$ helm fetch my-chart
```

>   This command does nothing if the same chart (name and version) already exists.

>   Using `--retry` is highly recommended in a CI/CD environment.

### Remove a chart

You can remove all the versions of a chart from a repository by running:

```shell
$ helm gcs remove my-chart my-repository
```

To remove a specific version, simply use the `--version` flag:

```shell
$ helm gcs remove my-chart my-repository --version 0.1.0
```

>   Don't forget to run `helm repo up` after you remove a chart.

## Troubleshooting

You can use the global flag `--debug`, or set `HELM_GCS_DEBUG=true` to get more informations. Please write an issue if you find any bug.
