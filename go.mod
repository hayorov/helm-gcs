module github.com/hayorov/helm-gcs

go 1.14

require (
	cloud.google.com/go/storage v1.6.0
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	google.golang.org/api v0.26.0
	helm.sh/helm v3.0.0-beta.3+incompatible
	k8s.io/client-go v11.0.0+incompatible // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
