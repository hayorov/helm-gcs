module github.com/hayorov/helm-gcs

go 1.13

require (
	cloud.google.com/go/storage v1.6.0
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v0.0.5
	google.golang.org/api v0.25.0
	gopkg.in/yaml.v2 v2.2.4 // indirect
	helm.sh/helm v3.0.0-beta.3+incompatible
	k8s.io/client-go v11.0.0+incompatible // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
