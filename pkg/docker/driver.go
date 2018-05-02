package docker

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron"

	"golang.org/x/sync/errgroup"

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

	if d.containerExists(file) {
		return fmt.Errorf("error: a logger for this container already exists: %s", file)
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

	if err := c.Log(pctx, config.url, config.Bulk.workers, c.indexName, config.tzpe, config.Bulk.actions, config.Bulk.size, config.Bulk.flushInterval, config.timeout); err != nil {
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

	if c.cron != nil {
		c.cron.Stop()
	}

	if c.pipeline.group != nil {
		c.logger.Info("closing pipeline")

		// Check whether any goroutines failed.
		if err := c.pipeline.group.Wait(); err != nil {
			c.logger.WithError(err).Error("pipeline wait group")
		}
	}

	if c.esClient != nil {
		c.logger.Info("stopping client")
		c.esClient.Stop()
	}

	return nil

}

func (d *Driver) containerExists(file string) bool {
	filename := path.Base(file)
	d.mu.Lock()
	if _, exists := d.logs[filename]; exists {
		return true
	}
	d.mu.Unlock()
	return false
}

// newContainer stores the container's configuration in memory
// and returns a pointer to the container
func (d *Driver) newContainer(ctx context.Context, file, containerID string) (*container, error) {

	filename := path.Base(file)

	c, err := newContainer(ctx, file, containerID)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
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
		return nil, fmt.Errorf("error: container not found for socket ID: %v", file)
	}

	return c, nil
}
