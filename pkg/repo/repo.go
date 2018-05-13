package repo

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/ghodss/yaml"
	"github.com/nouney/helm-gcs/pkg/gcs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

var (
	// ErrIndexOutOfDate occurs when trying to push a chart on a repository
	// that is being updated at the same time.
	ErrIndexOutOfDate = errors.New("index is out-of-date")

	// Debug is used to activate log output
	Debug bool
)

// Repo manages Helm repositories on Google Cloud Storage.
type Repo struct {
	entry               *repo.Entry
	indexFileURL        string
	indexFileGeneration int64
	gcs                 *storage.Client
}

// New creates a new Repo object
func New(path string, gcs *storage.Client) (*Repo, error) {
	indexFileURL, err := resolveReference(path, "index.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "resolve reference")
	}
	return &Repo{
		entry:        nil,
		indexFileURL: indexFileURL,
		gcs:          gcs,
	}, nil
}

// Load loads an existing repository known by Helm.
// Returns ErrNotFound if the repository is not found in helm repository entries.
func Load(name string, gcs *storage.Client) (*Repo, error) {
	entry, err := retrieveRepositoryEntry(name)
	if err != nil {
		return nil, errors.Wrap(err, "entry")
	}
	if entry == nil {
		return nil, fmt.Errorf("repository \"%s\" not found. Make sure you add it to helm", name)
	}

	indexFileURL, err := resolveReference(entry.URL, "index.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "resolve reference")
	}

	return &Repo{
		entry:        entry,
		indexFileURL: indexFileURL,
		gcs:          gcs,
	}, nil
}

// Create creates a new repository on GCS.
// This function is idempotent.
func Create(r *Repo) error {
	log := logger()
	log.Debugf("create a repository with index file at %s", r.indexFileURL)

	o, err := gcs.Object(r.gcs, r.indexFileURL)
	if err != nil {
		return errors.Wrap(err, "object")
	}

	_, err = o.NewReader(context.Background())
	if err == storage.ErrObjectNotExist {
		i := repo.NewIndexFile()
		return r.uploadIndexFile(i)
	} else if err == nil {
		log.Debugf("file %s already exists", r.indexFileURL)
		return nil
	}
	return err
}

// PushChart adds a chart into the repository.
//
// The index file on GCS will be updated and the file at "chartpath" will be uploaded to GCS.
// If the version of the chart is already indexed, it won't be uploaded unless "force" is set to true.
// The push will fail if the repository is updated at the same time, use "retry" to automatically reload
// the index of the repository.
func (r Repo) PushChart(chartpath string, force, retry bool) error {
	log := logger()
	i, err := r.indexFile()
	if err != nil {
		return errors.Wrap(err, "load index file")
	}

	log.Debugf("load chart \"%s\" (force=%t, retry=%t)", chartpath, force, retry)
	chart, err := chartutil.Load(chartpath)
	if err != nil {
		return errors.Wrap(err, "load chart")
	}

	log.Debugf("chart loaded: %s-%s", chart.Metadata.Name, chart.Metadata.Version)
	if i.Has(chart.Metadata.Name, chart.Metadata.Version) && !force {
		return fmt.Errorf("chart %s-%s already indexed. Use --force to still upload the chart", chart.Metadata.Name, chart.Metadata.Version)
	}

	if !i.Has(chart.Metadata.Name, chart.Metadata.Version) {
		err := r.updateIndexFile(i, chartpath, chart)
		if err == ErrIndexOutOfDate && retry {
			for err == ErrIndexOutOfDate {
				i, err = r.indexFile()
				if err != nil {
					return errors.Wrap(err, "load index file")
				}
				err = r.updateIndexFile(i, chartpath, chart)
			}
		}
		if err != nil {
			return errors.Wrap(err, "update index file")
		}
	}

	log.Debugf("upload file to GCS")
	err = r.uploadChart(chartpath)
	if err != nil {
		return errors.Wrap(err, "write chart")
	}
	return nil
}

// RemoveChart removes a chart from the repository
// If version is empty, all version will be deleted.
func (r Repo) RemoveChart(name, version string) error {
	log := logger()
	log.Debugf("removing chart %s-%s", name, version)

	index, err := r.indexFile()
	if err != nil {
		return errors.Wrap(err, "index")
	}

	vs, ok := index.Entries[name]
	if !ok {
		return fmt.Errorf("chart \"%s\" not found", name)
	}

	urls := []string{}
	for i, v := range vs {
		if version == "" || version == v.Version {
			log.Debugf("%s-%s will be deleted", name, v.Version)
			urls = append(urls, v.URLs...)
		}
		if version == v.Version {
			index.Entries[name] = append(vs[:i], vs[i+1:]...)
			break
		}
	}
	if version == "" || len(index.Entries[name]) == 0 {
		delete(index.Entries, name)
	}

	err = r.uploadIndexFile(index)
	if err != nil {
		return err
	}

	// Delete charts from GCS
	for _, url := range urls {
		o, err := gcs.Object(r.gcs, url)
		if err != nil {
			return errors.Wrap(err, "object")
		}

		log.Debugf("delete gcs file %s", url)
		err = o.Delete(context.Background())
		if err != nil {
			return errors.Wrap(err, "delete")
		}
	}
	return nil
}

// uploadIndexFile update the index file on GCS.
func (r Repo) uploadIndexFile(i *repo.IndexFile) error {
	log := logger()
	log.Debugf("push index file")
	i.SortEntries()
	o, err := gcs.Object(r.gcs, r.indexFileURL)
	if r.indexFileGeneration != 0 {
		log.Debugf("update condition: if generation = %d", r.indexFileGeneration)
		o = o.If(storage.Conditions{GenerationMatch: r.indexFileGeneration})
	}

	w := o.NewWriter(context.Background())
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
		gerr, ok := err.(*googleapi.Error)
		if ok && gerr.Code == 412 {
			return ErrIndexOutOfDate
		}
		return errors.Wrap(err, "close")
	}
	return nil
}

// indexFile retrieves the index file from GCS.
// It will also retrieve the generation number of the file, for optimistic locking.
func (r *Repo) indexFile() (*repo.IndexFile, error) {
	log := logger()
	log.Debugf("load index file \"%s\"", r.indexFileURL)

	// retrieve index file generation
	o, err := gcs.Object(r.gcs, r.indexFileURL)
	if err != nil {
		return nil, errors.Wrap(err, "object")
	}
	attrs, err := o.Attrs(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "attrs")
	}
	r.indexFileGeneration = attrs.Generation
	log.Debugf("index file generation: %d", r.indexFileGeneration)

	// get file
	reader, err := o.NewReader(context.Background())
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

// uploadChart pushes a chart into the repository.
func (r Repo) uploadChart(chartpath string) error {
	log := logger()
	f, err := os.Open(chartpath)
	if err != nil {
		return errors.Wrap(err, "open")
	}
	_, fname := filepath.Split(chartpath)
	chartURL, err := resolveReference(r.entry.URL, fname)
	if err != nil {
		return errors.Wrap(err, "resolve reference")
	}
	log.Debugf("upload file %s to gcs path %s", fname, chartURL)
	o, err := gcs.Object(r.gcs, chartURL)
	if err != nil {
		return errors.Wrap(err, "object")
	}
	w := o.NewWriter(context.Background())
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

func (r Repo) updateIndexFile(i *repo.IndexFile, chartpath string, chart *chart.Chart) error {
	log := logger()
	hash, err := provenance.DigestFile(chartpath)
	if err != nil {
		return errors.Wrap(err, "digest file")
	}
	_, fname := filepath.Split(chartpath)
	log.Debugf("indexing chart '%s-%s' as '%s' (base url: %s)", chart.Metadata.Name, chart.Metadata.Version, fname, r.entry.URL)
	i.Add(chart.GetMetadata(), fname, r.entry.URL, hash)
	return r.uploadIndexFile(i)
}

func resolveReference(base, p string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", errors.Wrap(err, "url parsing")
	}
	baseURL.Path = path.Join(baseURL.Path, p)
	return baseURL.String(), nil
}

func retrieveRepositoryEntry(name string) (*repo.Entry, error) {
	log := logger()
	helmHome := os.Getenv("HELM_HOME")
	if helmHome == "" {
		helmHome = environment.DefaultHelmHome
	}
	log.Debugf("helm home: %s", helmHome)
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

func logger() *logrus.Entry {
	l := logrus.New()
	level := logrus.InfoLevel
	if Debug || strings.ToLower(os.Getenv("HELM_GCS_DEBUG")) == "true" {
		level = logrus.DebugLevel
	}
	l.SetLevel(level)
	l.Formatter = &logrus.TextFormatter{}
	return logrus.NewEntry(l)
}
