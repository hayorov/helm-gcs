package repo

import (
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
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

var (
	Debug = false
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
	return &Repo{
		entry:        nil,
		indexFileURL: indexFileURL,
		gcs:          gcs,
	}, nil
}

/*
 * Load loads an existing repository known by Helm.
 *
 * Returns ErrNotFound if the repository is not found in helm repository entries.
 */
func Load(name string, gcs *storage.Client) (*Repo, error) {
	entry, err := retrieveRepositoryEntry(name)
	if err != nil {
		return nil, errors.Wrap(err, "entry")
	}
	if entry == nil {
		return nil, fmt.Errorf("repository \"%s\" not found. Make sure you add it to helm.")
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

/*
 * Create creates a new repository on GCS.
 *
 * Return an error if the repository already exists.
 */
func Create(r *Repo) error {
	log := logger()
	log.Debugf("create a repository with index file at %s", r.indexFileURL)
	_, err := gcs.NewReader(r.gcs, r.indexFileURL)
	if err == storage.ErrObjectNotExist {
		i := repo.NewIndexFile()
		return r.pushIndexFile(i)
	} else if err == nil {
		log.Debugf("file %s already exists", r.indexFileURL)
		return fmt.Errorf("index.yaml already exists")
	}
	return err
}

/*
 * AddChart adds a chart into the repository.
 *
 * If the chart already exists and "force" is false then nothing will happen.
 * Expects an already packaged chart (via "helm package").
 */
func (r Repo) PushChart(chartpath string, force bool) error {
	log := logger()
	log.Debugf("pushing chart %s into repository \"%s\" (%s) (force=%t)", chartpath, r.entry.Name, r.entry.URL, force)
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
		log.Debugf("indexing chart '%s-%s' as '%s' (base url: %s)", chart.Metadata.Name, chart.Metadata.Version, fname, r.entry.URL)
		i.Add(chart.GetMetadata(), fname, r.entry.URL, hash)
		err = r.pushIndexFile(i)
		if err != nil {
			return errors.Wrap(err, "write index")
		}
	} else if !force {
		log.Warnf("chart %s-%s already exists. Use --force if you still need to upload the chart", chart.Metadata.Name, chart.Metadata.Version)
		return nil
	}
	err = r.pushChart(chartpath)
	if err != nil {
		return errors.Wrap(err, "write chart")
	}
	return nil
}

/*
 * RemoveChart removes a chart from the repository
 *
 * If version is empty, all version will be deleted.
 */
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
	for i, v := range vs {
		if version == "" || version == v.Version {
			for _, url := range v.URLs {
				log.Debugf("delete version %s with url %s", v.Version, url)
				err := gcs.DeleteFile(r.gcs, url)
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
 * pushIndexFile update the index file on GCS.
 */
func (r Repo) pushIndexFile(i *repo.IndexFile) error {
	i.SortEntries()
	w, err := gcs.NewWriter(r.gcs, r.indexFileURL)
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
 * indexFile retrieves the index file from GCS.
 */
func (r Repo) indexFile() (*repo.IndexFile, error) {
	reader, err := gcs.NewReader(r.gcs, r.indexFileURL)
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
 * pushChart pushes a chart into the repository.
 */
func (r Repo) pushChart(chartpath string) error {
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
	log.Debugf("file %s will be uploaded to gcs path %s", fname, chartURL)
	w, err := gcs.NewWriter(r.gcs, chartURL)
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
