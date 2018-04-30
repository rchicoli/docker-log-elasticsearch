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
	bulkService map[int]elasticsearch.Bulk
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
					break
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
			case <-ctx.Done():
				c.logger.WithError(ctx.Err()).Error("closing read pipeline")
				return ctx.Err()
			}
			buf.Reset()
		}

		return nil
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
				c.logger.WithError(ctx.Err()).Error("closing parse pipeline")
				return ctx.Err()
			}

		}

		return nil
	})

	return nil
}

// Add adds messages to Elasticsearch Bulk Service
func (c *container) Log(ctx context.Context, workers int, indexName, tzpe string, actions, bulkSize int, flushInterval time.Duration) error {

	c.logger.Debug("starting pipeline: Log")

	var err error
	for i := 0; i < workers; i++ {
		workerID := i
		c.logger.WithField("workerID", workerID).Debug("starting worker")

		// one bulk service for each worker
		c.bulkService[workerID], err = elasticsearch.NewBulk(c.esClient)
		if err != nil {
			return err
		}
		// one ticker per worker
		// TODO: create a type worker
		ticker := time.NewTicker(flushInterval)

		c.pipeline.group.Go(func() error {

			defer func() {
				c.logger.WithField("workerID", workerID).Debug("closing worker")
				// commit any left messages in the queue
				c.flush(workerID, ctx)
				c.logger.Debug("stopping ticker")
				ticker.Stop()
			}()

			for {
				select {
				case doc, open := <-c.pipeline.outputCh:
					if !open {
						return nil
					}
					// c.logger.WithField("workerID", workerID).Debug("working number")

					c.bulkService[workerID].Add(indexName, tzpe, doc)

					if c.bulkService[workerID].CommitRequired(actions, bulkSize) {
						c.commit(workerID, ctx)
					}

				case <-ticker.C:
					// c.logger.WithFields(log.Fields{"ticker": ticker, "workerID": workerID}).Debug("ticking")
					c.flush(workerID, ctx)
				case <-ctx.Done():
					c.logger.WithError(ctx.Err()).Error("closing log pipeline")
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

func (c *container) flush(workerID int, ctx context.Context) {
	if c.bulkService[workerID].NumberOfActions() > 0 {
		c.commit(workerID, ctx)
	}
}

func (c *container) commit(workerID int, ctx context.Context) {

	// c.logger.WithField("size", c.esClient.EstimatedSizeInBytes()).Debug("estimed size in bytes")
	// c.logger.WithField("actions", c.esClient.NumberOfActions()).Debug("number of actions...")
	// c.logger.WithFields(log.Fields{"docs": c.bulkService[workerID].NumberOfActions(), "workerID": workerID}).Debug("bulking")

	bulkResponse, _, rerr, err := c.bulkService[workerID].Do(ctx)
	if err != nil || rerr {
		c.logger.WithError(err).Error("could not commit all messages to elasticsearch")

		// find out the reasons of the failure
		if responses := c.bulkService[workerID].Errors(bulkResponse); responses != nil {
			for _, response := range responses {
				for status, reason := range response {
					c.logger.WithFields(log.Fields{"reason": reason, "status": status}).Info("reason and status")
				}
			}
		}
	}
	// c.logger.WithFields(log.Fields{"took": took, "workerID": workerID}).Debug("bulk response time")
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
