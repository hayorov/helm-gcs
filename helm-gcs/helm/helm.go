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
	makeError := makeErrorFunc("helm.RetrieveRepositoryEntry")
	helmHome := os.Getenv("HELM_HOME")
	if helmHome == "" {
		helmHome = environment.DefaultHelmHome
	}
	debug("config dir: %s", helmHome)
	h := helmpath.Home(helmHome)
	repoFile, err := repo.LoadRepositoriesFile(h.RepositoryFile())
	if err != nil {
		return nil, makeError(err)
	}
	for _, r := range repoFile.Repositories {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, makeError(fmt.Errorf("repository %s does not exist", name))
}

func EmptyIndexFile() ([]byte, error) {
	makeError := makeErrorFunc("helm.EmptyIndexFile")
	f := repo.NewIndexFile()
	b, err := yaml.Marshal(f)
	if err != nil {
		return nil, makeError(err)
	}
	return b, nil
}

func LoadIndexFile(b []byte) (*repo.IndexFile, error) {
	makeError := makeErrorFunc("helm.LoadIndexFile")
	i := &repo.IndexFile{}
	if err := yaml.Unmarshal(b, i); err != nil {
		return nil, makeError(err)
	}
	i.SortEntries()
	return i, nil
}

var Debug bool

func debug(str string, args ...interface{}) {
	str = "helm: " + str
	if Debug {
		if len(args) == 0 {
			fmt.Println(str)
		} else {
			fmt.Printf(str+"\n", args...)
		}
	}
}

func makeErrorFunc(prefix string) func(error) error {
	return func(err error) error {
		return fmt.Errorf("%s: %s", prefix, err.Error())
	}
}
