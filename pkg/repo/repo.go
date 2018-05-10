package repo

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

type Repo struct {
	entry        *repo.Entry
	indexFileURL string
	gcs          *storage.Client
}

func New(path string, gcs *storage.Client) (*Repo, error) {
	indexFileURL, err := resolveReference(path, "index.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "resolve reference")
	}
	return &Repo{nil, indexFileURL, gcs}, nil
}

/*
 * Create creates a new repository on GCS.
 *
 * Return an error if the repository already exists.
 */
func Create(r *Repo) error {
	_, err := r.gcsReader(r.indexFileURL)
	if err == storage.ErrObjectNotExist {
		i := repo.NewIndexFile()
		return r.pushIndexFile(i)
	} else if err == nil {
		log.Printf("%s already exists.", r.indexFileURL)
	}
	return err
}

/*
 * Load loads an existing repository known by Helm.
 *
 * Returns an error if the repository is not found in helm repository entries.
 */
func Load(name string, gcs *storage.Client) (*Repo, error) {
	entry, err := retrieveRepositoryEntry(name)
	if err != nil {
		return nil, errors.Wrap(err, "entry")
	}
	if entry == nil {
		return nil, fmt.Errorf("repository \"%s\" not found in helm. Run `helm repo add %s gs://<BUCKET>/<PATH>`.", name, name)
	}
	indexFileURL, err := resolveReference(entry.URL, "index.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "resolve reference")
	}
	return &Repo{entry, indexFileURL, gcs}, nil
}

/*
 * AddChart adds a chart into the repository.
 *
 * If the chart already exists and "force" is false then nothing will happen.
 * Expects an already packaged chart (via "helm package").
 */
func (r Repo) PushChart(chartpath string, force bool) error {
	i, err := r.indexFile()
	if err != nil {
		return errors.Wrap(err, "index")
	}
	chart, err := chartutil.Load(chartpath)
	if err != nil {
		return errors.Wrap(err, "load chart")
	}
	// We do not need to update the index if the chart is already in it.
	// If force is true, then the chart will be uploaded even if already indexed.
	if i.Has(chart.Metadata.Name, chart.Metadata.Version) == false {
		hash, err := provenance.DigestFile(chartpath)
		if err != nil {
			return errors.Wrap(err, "digest file")
		}
		_, fname := filepath.Split(chartpath)
		i.Add(chart.GetMetadata(), fname, r.entry.URL, hash)
		err = r.pushIndexFile(i)
		if err != nil {
			return errors.Wrap(err, "write index")
		}
	} else if !force {
		log.Printf("chart %s-%s already exists. Use --force if you still need to upload the chart", chart.Metadata.Name, chart.Metadata.Version)
		return nil
	}
	err = r.pushChart(chartpath)
	if err != nil {
		return errors.Wrap(err, "write chart")
	}
	return nil
}

func (r Repo) RemoveChart(name, version string) error {
	index, err := r.indexFile()
	if err != nil {
		return errors.Wrap(err, "index")
	}
	vs, ok := index.Entries[name]
	if !ok {
		return fmt.Errorf("chart %s-%s does not exist in GCS", name, version)
	}
	for i, v := range vs {
		if version == "" || version == v.Version {
			for _, url := range v.URLs {
				err := r.gcsDelete(url)
				if err != nil {
					return errors.Wrap(err, "delete")
				}
			}
		}
		if version == v.Version {
			index.Entries[name] = append(vs[:i], vs[i+1:]...)
			break
		}
	}
	if version == "" {
		delete(index.Entries, name)
	}
	err = r.pushIndexFile(index)
	if err != nil {
		return err
	}
	return nil
}

/*
 * indexFile retrieves the index file from GCS.
 */
func (r Repo) indexFile() (*repo.IndexFile, error) {
	reader, err := r.gcsReader(r.indexFileURL)
	if err != nil {
		return nil, errors.Wrap(err, "reader")
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}
	defer reader.Close()
	i := &repo.IndexFile{}
	if err := yaml.Unmarshal(b, i); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	i.SortEntries()
	return i, nil
}

/*
 * pushIndexFile update the index file on GCS.
 */
func (r Repo) pushIndexFile(i *repo.IndexFile) error {
	i.SortEntries()
	w, err := r.gcsWriter(r.indexFileURL)
	if err != nil {
		return errors.Wrap(err, "writer")
	}
	b, err := yaml.Marshal(i)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	_, err = w.Write(b)
	if err != nil {
		return errors.Wrap(err, "write")
	}
	err = w.Close()
	if err != nil {
		return errors.Wrap(err, "close")
	}
	return nil
}

/*
 * pushChart pushes a chart into the repository.
 */
func (r Repo) pushChart(chartpath string) error {
	f, err := os.Open(chartpath)
	if err != nil {
		return errors.Wrap(err, "open")
	}
	_, fname := filepath.Split(chartpath)
	chartURL, err := resolveReference(r.entry.URL, fname)
	if err != nil {
		return errors.Wrap(err, "resolve reference")
	}
	w, err := r.gcsWriter(chartURL)
	if err != nil {
		return errors.Wrap(err, "writer")
	}
	_, err = io.Copy(w, f)
	if err != nil {
		return errors.Wrap(err, "copy")
	}
	err = w.Close()
	if err != nil {
		return errors.Wrap(err, "close")
	}
	return nil
}

/*
 * gcsWriter creates a new writer on GCS for the given path.
 */
func (r Repo) gcsWriter(path string) (io.WriteCloser, error) {
	bucket, path, err := splitPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	writer := r.gcs.Bucket(bucket).Object(path).NewWriter(ctx)
	return writer, nil
}

/*
 * gcsReader creates a new reader on GCS for the given path.
 */
func (r Repo) gcsReader(path string) (io.ReadCloser, error) {
	bucket, path, err := splitPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	reader, err := r.gcs.Bucket(bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (r Repo) gcsDelete(path string) error {
	bucket, path, err := splitPath(path)
	if err != nil {
		return errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	err = r.gcs.Bucket(bucket).Object(path).Delete(ctx)
	if err != nil {
		return errors.Wrap(err, "gcs")
	}
	return nil
}

func resolveReference(base, p string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", errors.Wrap(err, "url parsing")
	}
	baseURL.Path = path.Join(baseURL.Path, p)
	return baseURL.String(), nil
}

func splitPath(gcsurl string) (bucket string, path string, err error) {
	u, err := url.Parse(gcsurl)
	if err != nil {
		return
	}
	if u.Scheme != "gs" && u.Scheme != "gcs" {
		return "", "", errors.New(`incorrect url, should be "gs://bucket/path"`)
	}
	bucket = u.Host
	path = u.Path[1:]
	return
}

func retrieveRepositoryEntry(name string) (*repo.Entry, error) {
	helmHome := os.Getenv("HELM_HOME")
	if helmHome == "" {
		helmHome = environment.DefaultHelmHome
	}
	h := helmpath.Home(helmHome)
	repoFile, err := repo.LoadRepositoriesFile(h.RepositoryFile())
	if err != nil {
		return nil, errors.Wrap(err, "load")
	}
	for _, r := range repoFile.Repositories {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, errors.Wrapf(err, "repository \"%s\" does not exist", name)
}
