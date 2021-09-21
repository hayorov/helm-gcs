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
	"time"

	"cloud.google.com/go/storage"
	"github.com/ghodss/yaml"
	"github.com/hayorov/helm-gcs/pkg/gcs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	// ErrIndexOutOfDate occurs when trying to push a chart on a repository
	// that is being updated at the same time.
	ErrIndexOutOfDate = errors.New("index is out-of-date")

	// Debug is used to activate log output
	Debug bool
	log   = logger()
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
		return nil, errors.Wrap(err, "resolve index reference")
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
		return nil, errors.Wrap(err, "repo entry")
	}

	indexFileURL, err := resolveReference(entry.URL, "index.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "resolve index reference")
	}

	return &Repo{
		entry:        entry,
		indexFileURL: indexFileURL,
		gcs:          gcs,
	}, nil
}

// Create creates a new repository on GCS by uploading a blank index.yaml file.
// This function is idempotent.
func Create(r *Repo) error {
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
func (r Repo) PushChart(chartpath string, force, retry bool, public bool, publicURL string, bucketPath string, metadata map[string]string) error {
	i, err := r.indexFile()
	if err != nil {
		return errors.Wrap(err, "load index file")
	}

	log.Debugf("load chart \"%s\" (force=%t, retry=%t, public=%t)", chartpath, force, retry, public)
	chart, err := loader.Load(chartpath)
	if err != nil {
		return errors.Wrap(err, "load chart")
	}

	log.Debugf("chart loaded: %s-%s", chart.Metadata.Name, chart.Metadata.Version)
	if i.Has(chart.Metadata.Name, chart.Metadata.Version) && !force {
		return fmt.Errorf("chart %s-%s already indexed. Use --force to still upload the chart", chart.Metadata.Name, chart.Metadata.Version)
	}

	err = r.updateIndexFile(i, chartpath, chart, public, publicURL, bucketPath)
	if err == ErrIndexOutOfDate && retry {
		for err == ErrIndexOutOfDate {
			i, err = r.indexFile()
			if err != nil {
				return errors.Wrap(err, "load index file")
			}
			err = r.updateIndexFile(i, chartpath, chart, public, publicURL, bucketPath)
		}
	}
	if err != nil {
		return errors.Wrap(err, "update index file")
	}

	log.Debugf("upload file to GCS")
	err = r.uploadChart(chartpath, metadata)
	if err != nil {
		return errors.Wrap(err, "write chart")
	}
	return nil
}

// RemoveChart removes a chart from the repository
// If version is empty, all version will be deleted.
func (r Repo) RemoveChart(name, version string, retry bool) error {
	log.Debugf("removing chart %s-%s", name, version)

removeChart:
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
			vs[i] = vs[len(vs)-1]
			vs[len(vs)-1] = nil
			index.Entries[name] = vs[:len(vs)-1]
			break
		}
	}
	if version == "" || len(index.Entries[name]) == 0 {
		delete(index.Entries, name)
	}

	err = r.uploadIndexFile(index)
	if err == ErrIndexOutOfDate && retry {
		goto removeChart
	}

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
	log.Debugf("push index file")

	i.SortEntries()
	i.Generated = time.Now()

	o, err := gcs.Object(r.gcs, r.indexFileURL)
	if r.indexFileGeneration != 0 {
		log.Debugf("update condition: if generation = %d", r.indexFileGeneration)
		o = o.If(storage.Conditions{GenerationMatch: r.indexFileGeneration})
	}

	w := o.NewWriter(context.Background())
	if err != nil {
		return errors.Wrap(err, "writer")
	}
	// ensure index.yaml is not cached by GCS
	w.CacheControl = "no-cache, max-age=0, no-transform"

	// set the correct Content-Type ("text/yaml") for index.yaml file (solves issue #92)
	w.ContentType = "text/yaml"

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
func (r Repo) uploadChart(chartpath string, metadata map[string]string) error {
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

	w.Metadata = metadata

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

func (r Repo) updateIndexFile(i *repo.IndexFile, chartpath string, chart *chart.Chart, public bool, publicURL string, bucketPath string) error {
	hash, err := provenance.DigestFile(chartpath)
	if err != nil {
		return errors.Wrap(err, "generate chart file digest")
	}

	if bucketPath != "" {
		r.entry.URL = fmt.Sprintf("%s/%s", r.entry.URL, bucketPath)
	}

	url, err := getURL(r.entry.URL, public, publicURL)
	if err != nil {
		return errors.Wrap(err, "get chart base url")
	}

	_, fname := filepath.Split(chartpath)
	log.Debugf("indexing chart '%s-%s' as '%s' (base url: %s)", chart.Metadata.Name, chart.Metadata.Version, fname, url)

	// Need to remove current version of chart if there is any
	currentChart, _ := i.Get(chart.Metadata.Name, chart.Metadata.Version)
	if currentChart != nil {
		chartVersions := i.Entries[chart.Metadata.Name]
		for idx, ver := range chartVersions {
			if ver.Version == currentChart.Version {
				chartVersions[idx] = chartVersions[len(chartVersions)-1]
				chartVersions[len(chartVersions)-1] = nil
				i.Entries[chart.Metadata.Name] = chartVersions[:len(chartVersions)-1]
				break
			}
		}
	}

	i.Add(chart.Metadata, fname, url, hash)
	return r.uploadIndexFile(i)
}

func getURL(base string, public bool, publicURL string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if public && publicURL != "" {
		return publicURL, nil
	} else if public {
		return fmt.Sprintf("https://%s.storage.googleapis.com/%s", baseURL.Host, baseURL.Path), nil
	}
	return baseURL.String(), nil
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
	repoFilePath := envOr("HELM_REPOSITORY_CONFIG", helmpath.ConfigPath("repositories.yaml"))
	log.Debugf("helm repo file: %s", repoFilePath)

	repoFile, err := repo.LoadFile(repoFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "load repo file")
	}

	for _, r := range repoFile.Repositories {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, fmt.Errorf("repository \"%s\" does not exist", name)
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

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}
