package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/tonistiigi/fifo"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/extension/grok"

	protoio "github.com/gogo/protobuf/io"
)

const (
	name = "elasticsearchlog"
)

var l = log.New(os.Stderr, "", 0)

// Driver ...
type Driver struct {
	mu   *sync.Mutex
	logs map[string]*container
}

type pipeline struct {
	group    *errgroup.Group
	ctx      context.Context
	outputCh chan LogMessage
	inputCh  chan logdriver.LogEntry
	// stopCh   chan struct{}
}

type container struct {
	stream   io.ReadCloser
	info     logger.Info
	esClient elasticsearch.Client
	pipeline pipeline
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

// NewDriver ...
func NewDriver() *Driver {
	return &Driver{
		logs: make(map[string]*container),
		mu:   new(sync.Mutex),
	}
}

// StartLogging ...
func (d *Driver) StartLogging(file string, info logger.Info) error {

	// log.Printf("info: starting log: %s\n", file)
	filename := path.Base(file)

	// container's configuration is stored in memory
	d.mu.Lock()
	if _, exists := d.logs[filename]; exists {
		d.mu.Unlock()
		return fmt.Errorf("error: [%v] a logger for this container already exists", filename)
	}
	d.mu.Unlock()

	// check if need to be stored in memory
	ctx := context.Background()

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("error: opening logger fifo: %q", info.ContainerID)
	}

	d.mu.Lock()
	c := &container{stream: f, info: info}
	d.logs[filename] = c
	d.mu.Unlock()

	l.Printf("info: starting logger for containerID=[%v] and socket=[%v]\n", c.info.ContainerID, filename)

	config := defaultLogOpt()
	if err := config.validateLogOpt(c.info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
	}

	c.esClient, err = elasticsearch.NewClient(config.version, config.url, config.username, config.password, config.timeout, config.sniff, config.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	c.pipeline.group, c.pipeline.ctx = errgroup.WithContext(ctx)
	c.pipeline.inputCh = make(chan logdriver.LogEntry)
	c.pipeline.outputCh = make(chan LogMessage)
	// c.pipeline.stopCh = make(chan struct{})

	c.pipeline.group.Go(func() error {

		dec := protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
		defer func() {
			fmt.Printf("info: [%v] closing docker reader\n", c.info.ContainerID)
			dec.Close()
			close(c.pipeline.inputCh)
		}()

		var buf logdriver.LogEntry
		var err error

		for {
			if err = dec.ReadMsg(&buf); err != nil {
				if err == io.EOF {
					fmt.Printf("info: [%v] shutting down logger: %v\n", c.info.ContainerID, err)
					// c.stream.Close()
					return nil
				}
				if err != nil {
					// TODO: log only on debug mode
					// l.Printf("error: panicing [%v]: %v\n", c.info.ContainerID, err)
					// the connection has been closed
					// stop looping and closing the input channel
					// read /proc/self/fd/6: file already closed
					break
					// do not return, otherwise group.Go closes the pipeline
					// return err
				}

				dec = protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
			}

			// l.Printf("INFO pipe1 client: %#v\n", c.esClient)
			// l.Printf("INFO pipe1 line: %#v\n", string(buf.Line))

			// I guess this problem has been fixed with the break function above
			// test it again
			// BUG: (17.09.0~ce-0~debian) docker run command throws lots empty line messages
			if len(bytes.TrimSpace(buf.Line)) == 0 {
				// TODO: add log debug level
				// l.Printf("error trimming")
				continue
			}

			select {
			case c.pipeline.inputCh <- buf:
			case <-c.pipeline.ctx.Done():
				l.Printf("info: context done for pipe 1: %#v\n", c.pipeline.ctx.Err())
				return c.pipeline.ctx.Err()
			}
			buf.Reset()
		}

		return nil
	})

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

			// l.Printf("INFO pipe2: %#v\n", string(m.Line))

			// TODO: create a PR to grok upstream for parsing bytes
			// so that we avoid having to convert the message to string
			msg.GrokLine, msg.Line, err = groker.ParseLine(config.grokMatch, logMessage, m.Line)
			if err != nil {
				l.Printf("error: [%v] parsing log message: %v\n", c.info.ID(), err)
			}

			select {
			case c.pipeline.outputCh <- msg:
			case <-c.pipeline.ctx.Done():
				l.Printf("error: context done for pipe 2: %#v\n", c.pipeline.ctx.Err())
				return c.pipeline.ctx.Err()
			}

		}

		return nil
	})

	c.pipeline.group.Go(func() error {

		err := c.esClient.NewBulkProcessorService(c.pipeline.ctx, config.Bulk.workers, config.Bulk.actions, config.Bulk.size, config.Bulk.flushInterval, config.Bulk.stats)
		if err != nil {
			l.Printf("error creating bulk processor: %v\n", err)
		}

		// this was helpful to test if the pipeline has been closed successfully
		// newTicker := time.NewTicker(1 * time.Second)
		for doc := range c.pipeline.outputCh {
			// l.Printf("info: pipe3: %#v\n", string(doc.Line))
			// l.Printf("info: pipe3: %#v\n", doc.GrokLine)
			c.esClient.Add(config.index, config.tzpe, doc)
			select {
			case <-c.pipeline.ctx.Done():
				l.Printf("info context done for pipe 3: %#v\n", c.pipeline.ctx.Err())
				return c.pipeline.ctx.Err()
			// case <-newTicker.C:
			// 	l.Printf("info: still ticking")
			default:
			}
		}
		return nil
	})

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

	return nil
}

// StopLogging ...
// TODO: change api interface
func (d *Driver) StopLogging(file string) error {

	d.mu.Lock()
	filename := path.Base(file)
	c, exists := d.logs[filename]
	if !exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger not found for %v", filename)
	}
	delete(d.logs, file)
	d.mu.Unlock()

	l.Printf("info: stopping logger for containerID=[%v] and socket=[%v]\n", c.info.ContainerID, filename)

	if c.stream != nil {
		l.Printf("info: [%v] closing container stream\n", c.info.ID())
		c.stream.Close()
	}

	if c.pipeline.group != nil {
		l.Printf("info: [%v] closing pipeline: %v\n", c.info.ContainerID, c.pipeline)

		// Check whether any goroutines failed.
		if err := c.pipeline.group.Wait(); err != nil {
			l.Printf("error with pipeline [%v]: %v\n", filename, err)
		}
	}

	if c.esClient != nil {

		if err := c.esClient.Flush(); err != nil {
			l.Printf("error: flushing queue: %v", err)
		}

		l.Printf("info: closing client: %v", c.esClient)
		if err := c.esClient.Close(); err != nil {
			l.Printf("error: closing client connection: %v\n", err)
		}
		c.esClient.Stop()
	}

	// l.Printf("info: done stopping logger for containerID=[%v] and socket=[%v]\n", c.info.ContainerID, filename)

	return nil
}

// Name ...
func (d Driver) Name() string {
	return name
}
