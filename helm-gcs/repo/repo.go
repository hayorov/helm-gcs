package repo

import (
	"bytes"
	"strings"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/nouney/helm-gcs/helm-gcs/gcs"
	"github.com/nouney/helm-gcs/helm-gcs/helm"
	"k8s.io/helm/pkg/provenance"
	k8srepo "k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/chartutil"
)

type IndexFile = k8srepo.IndexFile

type Repo struct {
	*k8srepo.Entry
}

func LoadRepo(repoName string) (*Repo, error) {
	repo, err := helm.RetrieveRepositoryEntry(repoName)
	if err != nil {
		return nil, err
	}
	return &Repo{repo}, nil
}

func CreateRepo(path string) error {
	indexFile, err := helm.EmptyIndexFile()
	if err != nil {
		return err
	}
	w, err := gcs.NewWriter(path + "/index.yaml")
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(indexFile)
	_, err = io.Copy(w, buf)
	if err != nil {
		return err
	}
	return w.Close()
}

func (r Repo) LoadIndexFile() (*k8srepo.IndexFile, error) {
	reader, err := gcs.NewReader(r.Entry.URL + "/index.yaml")
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	i := &IndexFile{}
	if err := yaml.Unmarshal(b, i); err != nil {
		return nil, err
	}
	i.SortEntries()
	return i, nil
}


func (r Repo) AddChartFile(path string) error {
	i, err := r.LoadIndexFile()
	if err != nil {
		return err
	}
	chart, err := chartutil.Load(path)
	if err != nil {
		return err
	}
	if i.Has(chart.Metadata.Name, chart.Metadata.Version) == false {
		hash, err := provenance.DigestFile(path)
		if err != nil {
			return err
		}
		// Update index
		_, fname := filepath.Split(path)
		i.Add(chart.GetMetadata(), fname, r.Entry.URL, hash)
		err = r.WriteIndex(i)
		if err != nil {
			return err
		}
	}
	return r.WriteChart(path)
}

// RemoveChart removes a chart from the repository. If version is empty,
// all the versions will be removed.
func (r Repo) RemoveChart(name, version string) error {
	idx, err := r.LoadIndexFile()
	if err != nil {
		return err
	}
	versions, ok := idx.Entries[name]
	if !ok {
		return fmt.Errorf("chart %s-%s does not exist", name, version)
	}
	for i, ver := range versions {
		// delete all versions
		if version == "" {
			for _, url := range ver.URLs {
				if strings.HasPrefix(url, r.Entry.URL + "/" + name) {
					gcs.DeleteFile(url)
				}
			}
		} else if version == ver.Version {
			for _, url := range ver.URLs {
				if strings.HasPrefix(url, r.Entry.URL + "/" + name) {
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
		return err
	}
	return nil
}

func (r Repo) WriteIndex(i *IndexFile) error {
	i.SortEntries()
	// Upload index
	w, err := gcs.NewWriter(r.Entry.URL+"/index.yaml")
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		return err
	}
	return w.Close()
}

func (r Repo) WriteChart(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	_, fname := filepath.Split(path)
	writer, err := gcs.NewWriter(r.Entry.URL+"/"+fname)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, f)
	if err != nil {
		return err
	}
	return writer.Close()
}
