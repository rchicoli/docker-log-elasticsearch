package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron"

	"golang.org/x/sync/errgroup"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	log "github.com/sirupsen/logrus"
	"github.com/tonistiigi/fifo"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/extension/grok"

	protoio "github.com/gogo/protobuf/io"
)

const (
	name = "elasticsearchlog"
)

// Driver ...
type Driver struct {
	mu   *sync.Mutex
	logs map[string]*container
}

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
	ctx      context.Context
	group    *errgroup.Group
	inputCh  chan logdriver.LogEntry
	outputCh chan LogMessage
}

// LogMessage ...
type LogMessage struct {
	logdriver.LogEntry
	logger.Info

	GrokLine map[string]string
}

// MarshalJSON ...
func (l LogMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {

			// docker/daemon/logger/Info
			Config              map[string]string `json:"config,omitempty"`
			ContainerID         string            `json:"containerID,omitempty"`
			ContainerName       string            `json:"containerName,omitempty"`
			ContainerEntrypoint string            `json:"containerEntrypoint,omitempty"`
			ContainerArgs       []string          `json:"containerArgs,omitempty"`
			ContainerImageID    string            `json:"containerImageID,omitempty"`
			ContainerImageName  string            `json:"containerImageName,omitempty"`
			ContainerCreated    *time.Time        `json:"containerCreated,omitempty"`
			ContainerEnv        []string          `json:"containerEnv,omitempty"`
			ContainerLabels     map[string]string `json:"containerLabels,omitempty"`
			LogPath             string            `json:"logPath,omitempty"`
			DaemonName          string            `json:"daemonName,omitempty"`

			//  api/types/plugin/logdriver/LogEntry
			Line     string    `json:"message,omitempty"` // []byte to string
			Source   string    `json:"source"`
			TimeNano time.Time `json:"timestamp"` // int64 to Time
			Partial  bool      `json:"partial"`

			GrokLine map[string]string `json:"grok,omitempty"`
		}{
			Config:              l.Config,
			ContainerID:         l.ContainerID,
			ContainerName:       l.ContainerName,
			ContainerEntrypoint: l.ContainerEntrypoint,
			ContainerArgs:       l.ContainerArgs,
			ContainerImageID:    l.ContainerImageID,
			ContainerImageName:  l.ContainerImageName,
			ContainerCreated:    l.timeOmityEmpty(),
			ContainerEnv:        l.ContainerEnv,
			ContainerLabels:     l.ContainerLabels,
			LogPath:             l.LogPath,
			DaemonName:          l.DaemonName,

			GrokLine: l.GrokLine,

			Line:     string(l.Line),
			Source:   l.Source,
			TimeNano: time.Unix(0, l.TimeNano),
			Partial:  l.Partial,
		})

}

func (l LogMessage) timeOmityEmpty() *time.Time {
	if l.ContainerCreated.IsZero() {
		return nil
	}
	return &l.ContainerCreated
}

// NewDriver returns a pointer to driver
func NewDriver() *Driver {
	return &Driver{
		logs: make(map[string]*container),
		mu:   new(sync.Mutex),
	}
}

// StartLogging implements the docker plugin interface
func (d *Driver) StartLogging(file string, info logger.Info) error {

	filename := path.Base(file)
	log.WithField("fifo", file).Debug("created socket file for the logging")

	// container's configuration is stored in memory
	d.mu.Lock()
	if _, exists := d.logs[filename]; exists {
		d.mu.Unlock()
		return fmt.Errorf("error: a logger for this container already exists: %s", filename)
	}
	d.mu.Unlock()

	ctx := context.Background()

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("error: opening logger fifo: %q", info.ContainerID)
	}

	d.mu.Lock()
	c := &container{stream: f, info: info}
	d.logs[filename] = c
	d.mu.Unlock()

	c.logger = log.WithField("containerID", c.info.ContainerID)
	c.logger.WithField("socket", filename).Info("starting logging")

	config := defaultLogOpt()
	if err := config.validateLogOpt(c.info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
	}

	c.esClient, err = elasticsearch.NewClient(config.version, config.url, config.username, config.password, config.timeout, config.sniff, config.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	if indexFlag(config.index) {
		c.indexName = indexRegex(time.Now(), config.index)
		c.cron = cron.New()
		c.cron.AddFunc("@daily", func() {
			d.mu.Lock()
			c.indexName = indexRegex(time.Now(), config.index)
			d.mu.Unlock()
		})
		c.cron.Start()
	} else {
		c.indexName = config.index
	}

	c.pipeline.group, c.pipeline.ctx = errgroup.WithContext(ctx)
	c.pipeline.inputCh = make(chan logdriver.LogEntry)
	c.pipeline.outputCh = make(chan LogMessage)

	if err := d.Read(filename, config); err != nil {
		c.logger.WithError(err).Error("could not read line message")
	}

	if err := d.Parse(filename, config); err != nil {
		c.logger.WithError(err).Error("could not parse line message")
	}

	if err := d.Log(filename, config); err != nil {
		c.logger.WithError(err).Error("could not log to elasticsearch")
	}

	return nil
}

// Read reads messages from proto buffer
func (d *Driver) Read(filename string, config LogOpt) error {

	d.mu.Lock()
	c, exists := d.logs[filename]
	if !exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger not found for socket ID: %v", filename)
	}
	d.mu.Unlock()

	c.pipeline.group.Go(func() error {

		dec := protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
		defer func() {
			c.logger.Info("closing docker stream")
			dec.Close()
			close(c.pipeline.inputCh)
		}()

		var buf logdriver.LogEntry
		var err error

		for {
			if err = dec.ReadMsg(&buf); err != nil {
				if err == io.EOF {
					c.logger.Debug("shutting down reader eof")
					return nil
				}
				// the connection has been closed
				// stop looping and close the input channel
				// read /proc/self/fd/6: file already closed
				if strings.Contains(err.Error(), os.ErrClosed.Error()) {
					c.logger.WithError(err).Debug("shutting down fifo: closed by the writer")
					break
				}
				if err != nil {
					// the connection has been closed
					// stop looping and closing the input channel
					// read /proc/self/fd/6: file already closed
					c.logger.WithError(err).Debug("shutting down fifo")
					break
					// do not return, otherwise group.Go closes the pipeline
					// return err
				}

				dec = protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
			}

			// in case docker run command throws lots empty line messages
			if len(bytes.TrimSpace(buf.Line)) == 0 {
				c.logger.WithField("line", string(buf.Line)).Debug("trim space")
				continue
			}

			select {
			case c.pipeline.inputCh <- buf:
			case <-c.pipeline.ctx.Done():
				c.logger.WithError(c.pipeline.ctx.Err()).Error("closing read pipeline")
				return c.pipeline.ctx.Err()
			}
			buf.Reset()
		}

		return nil
	})

	return nil
}

// Stats shows metrics related to the bulk service
// func (d *Driver) Stats(filename string, config LogOpt) error {
// TODO: create metrics from stats
// d.pipeline.group.Go(func() error {
// 	stats := d.esClient.Stats()

// 	fields := log.Fields{
// 		"flushed":   stats.Flushed,
// 		"committed": stats.Committed,
// 		"indexed":   stats.Indexed,
// 		"created":   stats.Created,
// 		"updated":   stats.Updated,
// 		"succeeded": stats.Succeeded,
// 		"failed":    stats.Failed,
// 	}

// 	for i, w := range stats.Workers {
// 		fmt.Printf("Worker %d: Number of requests queued: %d\n", i, w.Queued)
// 		fmt.Printf("           Last response time       : %v\n", w.LastDuration)
// 		fields[fmt.Sprintf("w%d.queued", i)] = w.Queued
// 		fields[fmt.Sprintf("w%d.lastduration", i)] = w.LastDuration
// 	}
// })
// }

// Parse filters line messages
func (d *Driver) Parse(filename string, config LogOpt) error {

	d.mu.Lock()
	c, exists := d.logs[filename]
	if !exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger not found for socket ID: %v", filename)
	}
	d.mu.Unlock()

	c.pipeline.group.Go(func() error {
		defer close(c.pipeline.outputCh)

		groker, err := grok.NewGrok(config.grokMatch, config.grokPattern, config.grokPatternFrom, config.grokPatternSplitter, config.grokNamedCapture)
		if err != nil {
			return err
		}

		var logMessage string
		// custom log message fields
		msg := getLogOptFields(config.fields, c.info)

		for m := range c.pipeline.inputCh {

			logMessage = string(m.Line)

			// create message
			msg.Source = m.Source
			msg.Partial = m.Partial
			msg.TimeNano = m.TimeNano

			// TODO: create a PR to grok upstream for parsing bytes
			// so that we avoid having to convert the message to string
			msg.GrokLine, msg.Line, err = groker.ParseLine(config.grokMatch, logMessage, m.Line)
			if err != nil {
				c.logger.WithError(err).Error("could not parse line with grok")
			}

			select {
			case c.pipeline.outputCh <- msg:
			case <-c.pipeline.ctx.Done():
				c.logger.WithError(c.pipeline.ctx.Err()).Error("closing parse pipeline")
				return c.pipeline.ctx.Err()
			}

		}

		return nil
	})

	return nil
}

// Log sends messages to Elasticsearch Bulk Service
func (d *Driver) Log(filename string, config LogOpt) error {

	d.mu.Lock()
	c, exists := d.logs[filename]
	if !exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger not found for socket ID: %v", filename)
	}
	d.mu.Unlock()

	c.pipeline.group.Go(func() error {

		err := c.esClient.NewBulkProcessorService(c.pipeline.ctx, config.Bulk.workers, config.Bulk.actions, config.Bulk.size, config.Bulk.flushInterval, config.Bulk.stats)
		if err != nil {
			c.logger.WithError(err).Error("could not create bulk processor")
		}

		defer func() {
			if err := c.esClient.Flush(); err != nil {
				c.logger.WithError(err).Error("could not flush queue")
			}

			if err := c.esClient.Close(); err != nil {
				c.logger.WithError(err).Error("could not close client connection")
			}
			c.esClient.Stop()
		}()

		for doc := range c.pipeline.outputCh {

			c.esClient.Add(c.indexName, config.tzpe, doc)

			select {
			case <-c.pipeline.ctx.Done():
				c.logger.WithError(c.pipeline.ctx.Err()).Error("closing log pipeline")
				return c.pipeline.ctx.Err()
			default:
			}
		}
		return nil
	})

	return nil
}

// StopLogging implements the docker plugin interface
func (d *Driver) StopLogging(file string) error {

	// this is required for some environment like travis
	// otherwise the start and stop function are executed
	// too fast, even before messages are sent to the pipeline
	time.Sleep(1 * time.Second)

	d.mu.Lock()
	filename := path.Base(file)
	log.WithField("fifo", file).Debug("deleted socket file")

	c, exists := d.logs[filename]
	if !exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger not found for socket ID: %v", filename)
	}
	delete(d.logs, file)
	d.mu.Unlock()

	c.logger.WithField("socket", filename).Info("stopping logging")

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

// Name return the docker plugin name
func (d Driver) Name() string {
	return name
}
