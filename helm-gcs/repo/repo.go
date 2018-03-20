package repo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/nouney/helm-gcs/helm-gcs/gcs"
	"github.com/nouney/helm-gcs/helm-gcs/helm"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/provenance"
	k8srepo "k8s.io/helm/pkg/repo"
)

type IndexFile = k8srepo.IndexFile

type Repo struct {
	*k8srepo.Entry

	indexFileURL string
}

func LoadRepo(repoName string) (*Repo, error) {
	makeError := makeErrorFunc("repo.LoadRepo")
	repo, err := helm.RetrieveRepositoryEntry(repoName)
	if err != nil {
		return nil, makeError(err)
	}
	indexFileURL, err := resolveReference(repo.URL, "index.yaml")
	if err != nil {
		return nil, makeError(err)
	}
	return &Repo{repo, indexFileURL}, nil
}

func CreateRepo(path string) error {
	makeError := makeErrorFunc("repo.CreateRepo")
	indexFile, err := helm.EmptyIndexFile()
	if err != nil {
		return makeError(err)
	}
	indexFileURL, err := resolveReference(path, "/index.yaml")
	if err != nil {
		return makeError(err)
	}
	// Do not rewrite index.yaml if it already exists
	_, err = gcs.NewReader(indexFileURL)
	if err != nil {
		if err != gcs.ErrObjectNotExist {
			return makeError(err)
		}
	} else {
		debug("repository already initialized")
		return nil
	}
	// Create index.yaml
	w, err := gcs.NewWriter(indexFileURL)
	if err != nil {
		return makeError(err)
	}
	buf := bytes.NewBuffer(indexFile)
	_, err = io.Copy(w, buf)
	if err != nil {
		return makeError(err)
	}
	err = w.Close()
	if err != nil {
		return makeError(err)
	}
	return nil
}

func (r Repo) LoadIndexFile() (*k8srepo.IndexFile, error) {
	makeError := makeErrorFunc("repo.LoadIndexFile")
	reader, err := gcs.NewReader(r.indexFileURL)
	if err != nil {
		return nil, makeError(err)
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, makeError(err)
	}
	defer reader.Close()
	i := &IndexFile{}
	if err := yaml.Unmarshal(b, i); err != nil {
		return nil, makeError(err)
	}
	i.SortEntries()
	return i, nil
}

func (r Repo) AddChartFile(path string) error {
	makeError := makeErrorFunc("repo.AddChartFile")
	i, err := r.LoadIndexFile()
	if err != nil {
		return makeError(err)
	}
	chart, err := chartutil.Load(path)
	if err != nil {
		return makeError(err)
	}
	if i.Has(chart.Metadata.Name, chart.Metadata.Version) == false {
		hash, err := provenance.DigestFile(path)
		if err != nil {
			return makeError(err)
		}
		// Update index
		_, fname := filepath.Split(path)
		debug("indexing chart '%s-%s' as '%s' (base url: %s)", chart.Metadata.Name, chart.Metadata.Version, fname, r.Entry.URL)
		i.Add(chart.GetMetadata(), fname, r.Entry.URL, hash)
		err = r.WriteIndex(i)
		if err != nil {
			return makeError(err)
		}
	} else {
		debug("chart %s-%s already indexed", chart.Metadata.Name, chart.Metadata.Version)
	}
	err = r.WriteChart(path)
	if err != nil {
		return makeError(err)
	}
	return nil
}

// RemoveChart removes a chart from the repository. If version is empty,
// all the versions will be removed.
func (r Repo) RemoveChart(name, version string) error {
	makeError := makeErrorFunc("repo.RemoveChart")
	idx, err := r.LoadIndexFile()
	if err != nil {
		return makeError(err)
	}
	versions, ok := idx.Entries[name]
	if !ok {
		return makeError(fmt.Errorf("chart %s-%s does not exist", name, version))
	}
	for i, ver := range versions {
		// delete all versions
		if version == "" {
			for _, url := range ver.URLs {
				if strings.HasPrefix(url, r.Entry.URL+"/"+name) {
					gcs.DeleteFile(url)
				}
			}
		} else if version == ver.Version {
			for _, url := range ver.URLs {
				if strings.HasPrefix(url, r.Entry.URL+"/"+name) {
					gcs.DeleteFile(url)
				}
			}
			idx.Entries[name] = append(versions[:i], versions[i+1:]...)
			break
		}
	}
	if version == "" {
		delete(idx.Entries, name)
	}
	err = r.WriteIndex(idx)
	if err != nil {
		return makeError(err)
	}
	return nil
}

func (r Repo) WriteIndex(i *IndexFile) error {
	makeError := makeErrorFunc("repo.WriteIndex")
	i.SortEntries()
	// Upload index
	w, err := gcs.NewWriter(r.indexFileURL)
	if err != nil {
		return makeError(err)
	}
	b, err := yaml.Marshal(i)
	if err != nil {
		return makeError(err)
	}
	_, err = w.Write(b)
	if err != nil {
		return makeError(err)
	}
	err = w.Close()
	if err != nil {
		return makeError(err)
	}
	return nil
}

func (r Repo) WriteChart(path string) error {
	makeError := makeErrorFunc("repo.WriteChart")
	f, err := os.Open(path)
	if err != nil {
		return makeError(err)
	}
	_, fname := filepath.Split(path)
	chartURL, err := resolveReference(r.Entry.URL, fname)
	if err != nil {
		return makeError(err)
	}
	writer, err := gcs.NewWriter(chartURL)
	if err != nil {
		return makeError(err)
	}
	_, err = io.Copy(writer, f)
	if err != nil {
		return makeError(err)
	}
	err = writer.Close()
	if err != nil {
		return makeError(err)
	}
	return nil
}

func resolveReference(base, p string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	baseURL.Path = path.Join(baseURL.Path, p)
	return baseURL.String(), nil
}

var Debug bool

func debug(str string, args ...interface{}) {
	str = "repo: " + str
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
