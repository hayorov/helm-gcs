# helm-gcs
This is a Helm plugin that allows to use Google Cloud Storage as a (private) Helm repository.

## Usage

**You need to create a bucket on Google Cloud Storage.**

```shell
# Install the plugin
$ helm plugin install https://github.com/nouney/helm-gcs

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

# Fetch delete a chart
$ helm gcs remove myrepo/mychart
```
