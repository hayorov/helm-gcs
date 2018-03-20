# helm-gcs

`helm-gcs` is a [helm](https://github.com/kubernetes/helm) plugin that allows you to manage private helm repositories on Google Cloud Storage.

## Usage

**You need to create a bucket on Google Cloud Storage.**

```shell
# Install the plugin
$ helm plugin install https://github.com/nouney/helm-gcs --version 0.1.4

# Init a new repository
$ helm gcs init gs://bucket-name/path

# Add the repository to Helm
$ helm repo add myrepo gs://bucket-name/path

# Push a chart into the repository
$ helm gcs push mychart.tar.gz myrepo

# Update Helm cache
$ helm repo update

# Fetch the chart
$ helm fetch myrepo/mychart

# Delete a chart
$ helm gcs remove myrepo/mychart

# Update Helm cache
$ helm repo update
```
