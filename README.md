<p align="center">
	<img src="https://raw.githubusercontent.com/hayorov/helm-gcs/master/assets/helm-gcs-logo.png" alt="helm-gcs logo"/>
</p>

# helm-gcs

![Helm3 supported](https://img.shields.io/badge/Helm%203-supported-green)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/hayorov/helm-gcs)
[![Build Status](https://github.com/hayorov/helm-gcs/workflows/release/badge.svg)](https://github.com/hayorov/helm-gcs/releases/latest)

`helm-gcs` is a [helm](https://github.com/kubernetes/helm) plugin that allows you to manage private helm repositories on [Google Cloud Storage](https://cloud.google.com/storage/) aka buckets.

## Installation

Install the stable version:

```shell
$ helm plugin install https://github.com/hayorov/helm-gcs.git
```

Update to latest

```shell
$ helm plugin update gcs
```

Install a specific version:

```shell
$ helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.4.0
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

- Use the [application default credentials](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/)

- Use a service account via [`export GOOGLE_APPLICATION_CREDENTIALS=credentials.json` system variable](https://cloud.google.com/docs/authentication/getting-started)

- Use a temporary [OAuth 2.0 access token](https://developers.google.com/identity/protocols/oauth2) via `export GOOGLE_OAUTH_ACCESS_TOKEN=<MY_ACCESS_TOKEN>` environment variable. When used, plugin will ignore other authentification methods.

See [GCP documentation](https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application) for more information.

See also the section on working inside Terraform below.

### Create a repository

First, you need to [create a bucket on GCS](https://cloud.google.com/storage/docs/creating-buckets), which will be used by the plugin to store your charts.

Then you have to initialize a repository at a specific location in your bucket:

```shell
$ helm gcs init gs://your-bucket/path
```

> You can create a repository anywhere in your bucket.

> This command does nothing if a repository already exists at the given location.

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

Push the chart with additional option by providing metadata to the object :

```shell
$ helm gcs push my-chart-<semver>.tgz my-repository --metadata env=my-env,region=europe-west4
```

Push the chart with additional option by providing path inside bucket :

This would allow us to structure the content inside the bucket, and stores at `gs://your-bucket/my-application/my-chart-<semver>.tgz`

```shell
$ helm gcs push my-chart-<semver>.tgz my-repository --bucketPath=my-application
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

> This command does nothing if the same chart (name and version) already exists.

> Using `--retry` is highly recommended in a CI/CD environment.

### Remove a chart

You can remove all the versions of a chart from a repository by running:

```shell
$ helm gcs remove my-chart my-repository
```

To remove a specific version, simply use the `--version` flag:

```shell
$ helm gcs remove my-chart my-repository --version 0.1.0
```

> Don't forget to run `helm repo up` after you remove a chart.

## Troubleshooting

You can use the global flag `--debug`, or set `HELM_GCS_DEBUG=true` to get more informations. Please write an issue if you find any bug.

## Helm versions

Starting from 0.3 helm-gcs works with Helm 3, if you want to use it with Helm 2 please install the latest version that supports it

```shell
helm plugin install https://github.com/hayorov/helm-gcs.git --version 0.2.2 # helm 2 compatible
```

## Working with Terraform

It is possible to use the helm-gcs plugin along with the [Terraform Helm provider](https://registry.terraform.io/providers/hashicorp/helm/latest/docs), but you may need to pay special attention to your authentication configuration, and if you are using a remote execution environment such as Terraform Atlantis or Terraform Cloud you may need to perform some post-installation actions.

To use helm-gcs with the Terraform Helm Provider, first you will need to install it inside your Terraform module; for example if your Terraform files live in `${HOME}/src/terraform`, you would create a plugins directory there and install into it:

```shell
mkdir "${HOME}/src/terraform/helm_plugins" && \
  HELM_PLUGINS="${HOME}/src/terraform/helm_plugins" \
  helm plugin install https://github.com/hayorov/helm-gcs.git 
```

Note: if the OS/architecture of your local machine differs from the environment in which Terraform will actually execute (e.g. you are editing on macOS/arm but Terraform executes in Linux/amd64 via Atlantis or Terraform Cloud), you will need to manually run the installer script again in order to install the correct binary and set the `HELM_OS` and `HELM_ARCH` environment variables to override automatic detection of the local os and architecture:

```shell
HELM_PLUGIN_DIR="${HOME}/src/terraform/helm_plugins/helm-gcs.git" \
  HELM_OS="linux" \
  HELM_ARCH="x86_64" \
  "${HOME}/src/terraform/helm_plugins/helm-gcs.git/scripts/install.sh"
```

Once the plugin is installed, add its parent directory to the `plugins_path` attribute of your Helm provider definition:

```hcl
provider "helm" {
   kubernetes {
    host                   = "https://${google_container_cluster.default.endpoint}"
    token                  = data.google_client_config.provider.access_token
    cluster_ca_certificate = base64decode(google_container_cluster.default.master_auth[0].cluster_ca_certificate)
  }
  plugins_path = "${path.module}/helm_plugins"
}
```

With this in place you should be able to install Helm charts from repositories in GCS:

```hcl
resource "helm_release" "my_chart" {
  name       = "my-chart"
  chart      = "my-chart"
  repository = "gs://your-bucket/path"
  timeout    = 600
  replace    = true
  atomic     = true
}
```

### Authentication inside Terraform

Terraform's [Google Cloud Platform Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs) adds an option to the [default](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/) resolution method to determine your authentication credentials: if the environment variable `GOOGLE_CREDENTIALS` is set, it will attempt to read the JSON key file out of that environment variable. (Details [here](https://registry.terraform.io/providers/hashicorp/helm/latest/docs#authentication)) This is most commonly used with hosted Terraform execution environments such as Terraform Atlantis and Terraform Cloud.

If the `GOOGLE_CREDENTIALS` environment variable is set, helm-gcs will attempt to use its value preferentially as its service account credentials! To disable this behavior and fall back to the defaults, set the environment variable `HELM_GCS_IGNORE_TERRAFORM_CREDS` to `true` in your execution workspace.
