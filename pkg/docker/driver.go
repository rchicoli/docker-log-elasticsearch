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
	ctx  context.Context
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

	ctx := context.Background()

	c, err := d.newContainer(ctx, file)
	if err != nil {
		return err
	}
	c.info = info
	c.logger = log.WithField("containerID", info.ContainerID)

	config := newConfiguration()
	if err := config.validateLogOpt(c.info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
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
	c.pipeline.inputCh = make(chan logdriver.LogEntry)
	c.pipeline.outputCh = make(chan LogMessage)

	if err := d.Read(pctx, file); err != nil {
		c.logger.WithError(err).Error("could not read line message")
	}

	if err := d.Parse(pctx, file, config.fields, config.grokMatch, config.grokPattern, config.grokPatternFrom, config.grokPatternSplitter, config.grokNamedCapture); err != nil {
		c.logger.WithError(err).Error("could not parse line message")
	}

	if err := d.Log(pctx, file, config); err != nil {
		c.logger.WithError(err).Error("could not log to elasticsearch")
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
