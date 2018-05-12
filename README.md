# helm-gcs

`helm-gcs` is a [helm](https://github.com/kubernetes/helm) plugin that allows you to manage private helm repositories on Google Cloud Storage.

## Installation

Install the latest version:
```shell
$ helm plugin install https://github.com/nouney/helm-gcs
```

Install a specific version:
```shell
$ helm plugin install https://github.com/nouney/helm-gcs --version 0.2.0
```

## Getting started

### Authentification

See [the GCP documentation](https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application).

### Create a repository

First, you need to [create a bucket on GCS](https://cloud.google.com/storage/docs/creating-buckets). It will be used by the plugin to store your charts.

Then you have to initialize a repository at a specific location in your bucket:

```shell
$ helm gcs init gs://your-bucket/path
```

>   You can create a repository anywhere in your bucket.

>   CI/CD: `init` is idempotent.

You can now add the repository:
```shell
$ helm repo add my-repository gs://your-bucket/path
```

### Push a chart

Package the chart if it's not yet:
```shell
$ helm package chartDir
```
This will create a file `chartDir-<semver>.tgz`.

Now, to push the chart to the repository `my-repository`:

```shell
$ helm gcs push chartDir-<semver>.tgz my-repository
```

If you got this error:
```shell
Error: update index file: index is out-of-date
```

That means that someone/something updated the same repository, at the same time as you. You just need to execute the command again or, next time, use the `--retry` flag to automatically retry to push the chart.

>   Don't forget to run `helm repo up` after you push a chart.

>   CI/CD: `push` is idempotent. It is recommended to use the flag `--retry`, otherwise 

### Remove a chart

You can remove all the versions of a chart from a repository by running:

```shell
$ helm gcs remove myChart my-repository
```

To remove a specific version, simply use the `--version` flag:

```shell
$ helm gcs remove myChart my-repository --version 0.1.0
```

>   Don't forget to run `helm repo up` after you remove a chart.

## Costs

## Troubleshooting

You can use the global flag `--debug` to get more informations. Please write an issue if you find a bug.