module github.com/hayorov/helm-gcs

go 1.13

require (
	cloud.google.com/go/storage v1.0.0
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	google.golang.org/api v0.11.1-0.20191026000714-8038894d941c
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843 // indirect
	k8s.io/client-go v11.0.0+incompatible // indirect
	k8s.io/helm v2.15.1+incompatible
)
