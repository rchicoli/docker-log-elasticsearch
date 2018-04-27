package docker

import (
	"context"
	"fmt"
	"io"
	"path"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/robfig/cron"
	"github.com/tonistiigi/fifo"
	"golang.org/x/sync/errgroup"
)

type container struct {
	cron      *cron.Cron
	esClient  elasticsearch.Client
	indexName string
	info      logger.Info
	logger    *log.Entry
	pipeline  pipeline
	stream    io.ReadCloser
}

type pipeline struct {
	group    *errgroup.Group
	inputCh  chan logdriver.LogEntry
	outputCh chan LogMessage
}

// newContainer stores the container's configuration in memory
// and returns a pointer to the container
func (d *Driver) newContainer(ctx context.Context, file string) (*container, error) {

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
	c := &container{stream: f}
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
