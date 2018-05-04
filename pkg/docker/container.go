package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/extension/grok"
	"github.com/robfig/cron"
	"github.com/tonistiigi/fifo"
	"golang.org/x/sync/errgroup"
)

type container struct {
	// bulkService map[int]*BulkWorker
	cron      *cron.Cron
	esClient  elasticsearch.Client
	indexName string
	logger    *log.Entry
	pipeline  pipeline
	stream    io.ReadCloser
}

type pipeline struct {
	// commitCh chan struct{}
	group    *errgroup.Group
	inputCh  chan logdriver.LogEntry
	outputCh chan LogMessage
}

// Processor interface
// type Processor interface {
// 	Read(ctx context.Context) error
// 	Parse(ctx context.Context, info logger.Info, fields, grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter string, grokNamedCapture bool) error
// 	Log(ctx context.Context, workers int, indexName, tzpe string, actions, bulkSize int, flushInterval, timeout time.Duration) error
// }

// newContainer stores the container's configuration in memory
// and returns a pointer to the container
func newContainer(ctx context.Context, file, containerID string) (*container, error) {

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return nil, fmt.Errorf("could not open fifo: %q", err)
	}

	return &container{
		// bulkService: make(map[int]*BulkWorker),
		stream: f,
		logger: log.WithField("containerID", containerID),
		pipeline: pipeline{
			// commitCh: make(chan struct{}),
			inputCh:  make(chan logdriver.LogEntry),
			outputCh: make(chan LogMessage),
		},
	}, nil
}

// Read reads messages from proto buffer
func (c *container) Read(ctx context.Context) error {

	c.logger.Debug("starting pipeline: Read")

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
					// shutdown gracefully all pipelines
					return nil
				}
				if err != nil {
					// the connection has been closed
					// stop looping and closing the input channel
					// read /proc/self/fd/6: file already closed
					c.logger.WithError(err).Debug("shutting down fifo")
					return err
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
			case <-ctx.Done():
				c.logger.WithError(ctx.Err()).Error("closing read pipeline: Read")
				return ctx.Err()
			}
			buf.Reset()
		}

	})

	return nil
}

// Parse filters line messages
func (c *container) Parse(ctx context.Context, info logger.Info, fields, grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter string, grokNamedCapture bool) error {

	c.logger.Debug("starting pipeline: Parse")

	c.pipeline.group.Go(func() error {
		defer close(c.pipeline.outputCh)

		groker, err := grok.NewGrok(grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter, grokNamedCapture)
		if err != nil {
			return err
		}

		var logMessage string
		// custom log message fields
		msg := getLogMessageFields(fields, info)

		for m := range c.pipeline.inputCh {

			logMessage = string(m.Line)

			// create message
			msg.Source = m.Source
			msg.Partial = m.Partial
			msg.TimeNano = m.TimeNano

			// TODO: create a PR to grok upstream for parsing bytes
			// so that we avoid having to convert the message to string
			msg.GrokLine, msg.Line, err = groker.ParseLine(grokMatch, logMessage, m.Line)
			if err != nil {
				c.logger.WithError(err).Error("could not parse line with grok")
			}

			select {
			case c.pipeline.outputCh <- msg:
			case <-ctx.Done():
				c.logger.WithError(ctx.Err()).Error("closing parse pipeline: Parse")
				return ctx.Err()
			}

		}

		return nil
	})

	return nil
}

// Log sends messages to Elasticsearch Bulk Service
func (c *container) Log(ctx context.Context, workers, actions, size int, flushInterval, timeout time.Duration, stats bool, indexName, tzpe string) error {

	c.logger.Debug("starting pipeline: Log")

	c.pipeline.group.Go(func() error {

		err := c.esClient.NewBulkProcessorService(
			ctx,
			workers,
			actions,
			size,
			flushInterval,
			timeout,
			false,
			c.logger,
		)
		if err != nil {
			c.logger.WithError(err).Error("could not create bulk processor")
			return err
		}

		defer func() {
			if err := c.esClient.Flush(); err != nil {
				c.logger.WithError(err).Error("could not flush queue")
			}

			if err := c.esClient.Close(); err != nil {
				c.logger.WithError(err).Error("could not close bulk processor")
			}
		}()

		for doc := range c.pipeline.outputCh {

			c.esClient.Add(indexName, tzpe, doc)

			select {
			case <-ctx.Done():
				c.logger.WithError(ctx.Err()).Error("closing log pipeline")
				return ctx.Err()
			default:
			}
		}
		return nil
	})

	return nil
}

// BulkWorkerService interface
type BulkWorkerService interface {
	Flush(ctx context.Context)
	Commit(ctx context.Context)
}

// BulkWorker provides a Bulk Processor
type BulkWorker struct {
	elasticsearch.Bulk
	logger *log.Entry
	ticker *time.Ticker
}

func newWorker(client elasticsearch.Client, logEntry *log.Entry, actions, workerID int, flushInterval, timeout time.Duration) (*BulkWorker, error) {
	bulkService, err := elasticsearch.NewBulk(client, timeout, actions)
	if err != nil {
		return nil, err
	}
	return &BulkWorker{
		Bulk:   bulkService,
		logger: logEntry.WithField("workerID", workerID),
		ticker: time.NewTicker(flushInterval),
	}, nil
}

// Flush checks if there are actions to be commited
// before sending them to elasticsearch
func (b BulkWorker) Flush(ctx context.Context) {
	if b.NumberOfActions() > 0 {
		b.Commit(ctx)
	}
}

// Commit sends all messages to elasticsearch
func (b *BulkWorker) Commit(ctx context.Context) {

	// b.logger.WithField("size", c.esClient.EstimatedSizeInBytes()).Debug("estimed size in bytes")
	// b.logger.WithField("actions", c.esClient.NumberOfActions()).Debug("number of actions...")
	// b.logger.WithFields(log.Fields{"docs": b.NumberOfActions(), "workerID": workerID}).Debug("bulking")

	bulkResponse, _, rerr, err := b.Do(ctx)
	if rerr {
		// find out the reasons of the failure
		if responses := b.Errors(bulkResponse); responses != nil {
			for _, response := range responses {
				for range response {
					for status, reason := range response {
						if status == 429 {
							b.logger.WithFields(log.Fields{"reason": reason, "status": status}).Info("resending request")
						} else {
							b.logger.WithFields(log.Fields{"reason": reason, "status": status}).Info("response error message and status code")
						}
					}
				}
			}
			return
		}
	}
	if err != nil {
		b.logger.WithError(err).Error("could not send all messages to elasticsearch")
	}
	// b.logger.WithFields(log.Fields{"took": took}).Debug("bulk response time")
}

// Stats shows metrics related to the bulk service
// func (d *Driver) Stats(filename string, config Configuration) error {
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
