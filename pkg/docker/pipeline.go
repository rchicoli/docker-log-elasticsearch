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

// Log sends messages to Elasticsearch Bulk Service
func (c *container) Log(ctx context.Context, workers, actions, size int, flushInterval time.Duration, stats bool, indexName, tzpe string) error {

	c.logger.Debug("starting pipeline: Log")

	c.pipeline.group.Go(func() error {

		err := c.esClient.NewBulkProcessorService(ctx, workers, actions, size, flushInterval, stats)
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
