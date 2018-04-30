package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/extension/grok"
	"github.com/robfig/cron"
	"golang.org/x/sync/errgroup"
)

type container struct {
	cron        *cron.Cron
	esClient    elasticsearch.Client
	bulkService map[int]*BulkWorker
	indexName   string
	logger      *log.Entry
	pipeline    pipeline
	stream      io.ReadCloser
}

type pipeline struct {
	group    *errgroup.Group
	inputCh  chan logdriver.LogEntry
	outputCh chan LogMessage
	commitCh chan struct{}
	ticker   *time.Ticker
}

type Processor interface {
	Read(ctx context.Context) error
	Parse(ctx context.Context, info logger.Info, fields, grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter string, grokNamedCapture bool) error
	Log(ctx context.Context, workers int, indexName, tzpe string, actions, bulkSize int, flushInterval time.Duration) error
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
					// break
					// shutdown grafully all pipelines
					return nil
				}
				if err != nil {
					// the connection has been closed
					// stop looping and closing the input channel
					// read /proc/self/fd/6: file already closed
					c.logger.WithError(err).Debug("shutting down fifo")
					// break
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

		// return nil
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

type BulkWorker struct {
	elasticsearch.Bulk
	logger *log.Entry
	ticker *time.Ticker
}

func newWorker(client elasticsearch.Client, logEntry *log.Entry, workerID int, flushInterval time.Duration) (*BulkWorker, error) {
	bulkService, err := elasticsearch.NewBulk(client)
	if err != nil {
		return nil, err
	}
	return &BulkWorker{
		Bulk:   bulkService,
		logger: logEntry.WithField("workerID", workerID),
		ticker: time.NewTicker(flushInterval),
	}, nil
}

// Add adds messages to Elasticsearch Bulk Service
func (c *container) Log(ctx context.Context, workers int, indexName, tzpe string, actions, bulkSize int, flushInterval time.Duration) error {

	c.logger.Debug("starting pipeline: Log")

	var err error
	for i := 0; i < workers; i++ {
		workerID := i

		// one bulk service for each worker
		c.bulkService[workerID], err = newWorker(c.esClient, c.logger, workerID, flushInterval)
		if err != nil {
			return err
		}

		c.bulkService[workerID].logger.Debug("starting worker")
		c.pipeline.group.Go(func() error {

			defer func() {
				c.bulkService[workerID].logger.Debug("closing worker")
				// commit any left messages in the queue
				c.bulkService[workerID].Flush(ctx)
				c.logger.Debug("stopping ticker")
				c.bulkService[workerID].ticker.Stop()
			}()

			for {
				select {
				case doc, open := <-c.pipeline.outputCh:
					if !open {
						return nil
					}
					c.bulkService[workerID].Add(indexName, tzpe, doc)

					if c.bulkService[workerID].CommitRequired(actions, bulkSize) {
						c.bulkService[workerID].Commit(ctx)
					}

				case <-c.bulkService[workerID].ticker.C:
					// c.bulkService[workerID].logger.WithField("ticker", c.bulkService[workerID].ticker).Debug("ticking")
					c.bulkService[workerID].Flush(ctx)
				case <-ctx.Done():
					c.bulkService[workerID].logger.WithError(ctx.Err()).Error("closing log pipeline: Log")
					return ctx.Err()
					// commit has to be in the same goroutine
					// because of reset is called in the Do() func
					// case c.pipeline.commitCh <- struct{}{}:
				}
			}
		})
	}

	return nil
}

type BulkWorkerService interface {
	Flush(ctx context.Context)
	Commit(ctx context.Context)
}

func (b BulkWorker) Flush(ctx context.Context) {
	if b.NumberOfActions() > 0 {
		b.Commit(ctx)
	}
}

func (b *BulkWorker) Commit(ctx context.Context) {

	// b.logger.WithField("size", c.esClient.EstimatedSizeInBytes()).Debug("estimed size in bytes")
	// b.logger.WithField("actions", c.esClient.NumberOfActions()).Debug("number of actions...")
	// b.logger.WithFields(log.Fields{"docs": c.bulkService[workerID].NumberOfActions(), "workerID": workerID}).Debug("bulking")

	bulkResponse, _, rerr, err := b.Do(ctx)
	if err != nil || rerr {
		b.logger.WithError(err).Error("could not commit all messages to elasticsearch")

		// find out the reasons of the failure
		if responses := b.Errors(bulkResponse); responses != nil {
			for _, response := range responses {
				for status, reason := range response {
					b.logger.WithFields(log.Fields{"reason": reason, "status": status}).Info("reason and status")
				}
			}
		}
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
