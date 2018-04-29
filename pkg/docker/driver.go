package docker

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron"
	"github.com/tonistiigi/fifo"

	"golang.org/x/sync/errgroup"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/regex"
)

const (
	name = "elasticsearchlog"
)

// Driver ...
type Driver struct {
	mu   *sync.Mutex
	logs map[string]*container
}

// NewDriver returns a pointer to driver
func NewDriver() *Driver {
	return &Driver{
		logs: make(map[string]*container),
		mu:   new(sync.Mutex),
	}
}

// Name return the docker plugin name
func (d *Driver) Name() string {
	return name
}

// StartLogging implements the docker plugin interface
func (d *Driver) StartLogging(file string, info logger.Info) error {

	config := newConfiguration()
	if err := config.validateLogOpt(info.Config); err != nil {
		return err
	}

	ctx := context.Background()

	c, err := d.newContainer(ctx, file, info.ContainerID)
	if err != nil {
		return err
	}

	c.esClient, err = elasticsearch.NewClient(config.version, config.url, config.username, config.password, config.timeout, config.sniff, config.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	// org.elasticsearch.indices.InvalidIndexNameException: ... must be lowercase
	c.indexName = strings.ToLower(regex.ParseDate(time.Now(), config.index))
	if regex.IsValid(config.index) {
		c.cron = cron.New()
		c.cron.AddFunc("@daily", func() {
			d.mu.Lock()
			c.indexName = strings.ToLower(regex.ParseDate(time.Now(), config.index))
			d.mu.Unlock()
		})
		c.cron.Start()
	}

	var pctx context.Context
	c.pipeline.group, pctx = errgroup.WithContext(ctx)

	if err := c.Read(pctx); err != nil {
		c.logger.WithError(err).Error("could not read line message")
		return err
	}

	if err := c.Parse(pctx, info, config.fields, config.grokMatch, config.grokPattern, config.grokPatternFrom, config.grokPatternSplitter, config.grokNamedCapture); err != nil {
		c.logger.WithError(err).Error("could not parse line message")
		return err
	}

	if err := c.Log(pctx, config.Bulk.workers, c.indexName, config.tzpe); err != nil {
		c.logger.WithError(err).Error("could not log to elasticsearch")
		return err
	}

	if err := c.Commit(pctx, config.Bulk.actions, config.Bulk.size, config.Bulk.flushInterval); err != nil {
		c.logger.WithError(err).Error("could not log to elasticsearch")
		return err
	}

	return nil

}

// StopLogging implements the docker plugin interface
func (d *Driver) StopLogging(file string) error {

	// this is required for some environment like travis
	// otherwise the start and stop function are executed
	// too fast, even before messages are sent to the pipeline
	time.Sleep(1 * time.Second)

	c, err := d.getContainer(file)
	if err != nil {
		return err
	}

	filename := path.Base(file)
	d.mu.Lock()
	delete(d.logs, filename)
	d.mu.Unlock()

	c.logger.WithField("fifo", file).Debug("removing fifo file")

	if c.stream != nil {
		c.logger.Info("closing container stream")
		c.stream.Close()
	}

	if c.pipeline.group != nil {
		c.logger.Info("closing pipeline")

		// Check whether any goroutines failed.
		if err := c.pipeline.group.Wait(); err != nil {
			c.logger.WithError(err).Error("pipeline wait group")
		}
	}

	if c.cron != nil {
		c.cron.Stop()
	}

	// if c.esClient != nil {
	//	close client connection on last pipeline
	// }

	return nil

}

// newContainer stores the container's configuration in memory
// and returns a pointer to the container
func (d *Driver) newContainer(ctx context.Context, file, containerID string) (*container, error) {

	filename := path.Base(file)
	log.WithField("fifo", file).Debug("created fifo file")

	d.mu.Lock()
	if _, exists := d.logs[filename]; exists {
		return nil, fmt.Errorf("error: a logger for this container already exists: %s", filename)
	}
	d.mu.Unlock()

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return nil, fmt.Errorf("could not open fifo: %q", err)
	}

	d.mu.Lock()
	c := &container{
		stream: f,
		logger: log.WithField("containerID", containerID),
		pipeline: pipeline{
			commitCh: make(chan struct{}),
			inputCh:  make(chan logdriver.LogEntry),
			outputCh: make(chan LogMessage),
		},
	}
	d.logs[filename] = c
	d.mu.Unlock()

	return c, nil
}

// getContainer retrieves the container's configuration from memory
func (d *Driver) getContainer(file string) (*container, error) {

	filename := path.Base(file)

	d.mu.Lock()
	defer d.mu.Unlock()

	c, exists := d.logs[filename]
	if !exists {
		return nil, fmt.Errorf("error: logger not found for socket ID: %v", file)
	}

	return c, nil
}
