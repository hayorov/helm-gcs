package helm

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

func RetrieveRepositoryEntry(name string) (*repo.Entry, error) {
	helmHome := os.Getenv("HELM_HOME")
	if helmHome == "" {
		helmHome = environment.DefaultHelmHome
	}
	h := helmpath.Home(helmHome)
	repoFile, err := repo.LoadRepositoriesFile(h.RepositoryFile())
	if err != nil {
		return nil, err
	}
	for _, r := range repoFile.Repositories {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("repository %s does not exist", name)
}

func EmptyIndexFile() ([]byte, error) {
	f := repo.NewIndexFile()
	return yaml.Marshal(f)
}

func LoadIndexFile(b []byte) (*repo.IndexFile, error) {
	i := &repo.IndexFile{}
	if err := yaml.Unmarshal(b, i); err != nil {
		return nil, err
	}
	i.SortEntries()
	return i, nil
}
