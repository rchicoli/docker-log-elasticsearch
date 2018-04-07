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
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/tonistiigi/fifo"

	"github.com/rchicoli/docker-log-elasticsearch/internal/pkg/errgroup"
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
	stopCh   chan struct{}
}

type container struct {
	stream   io.ReadCloser
	info     logger.Info
	esClient elasticsearch.Client
	pipeline pipeline
	config   *LogOpt
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

	// container's configuration is stored in memory
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("error: [%v] a logger for this container already exists", info.ContainerID)
	}
	d.mu.Unlock()

	ctx := context.Background()

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("error: opening logger fifo: %q", file)
	}

	d.mu.Lock()
	d.logs[file] = &container{stream: f, info: info}
	d.mu.Unlock()

	d.logs[file].config = defaultLogOpt()
	if err := d.logs[file].config.validateLogOpt(d.logs[file].info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
	}

	d.logs[file].esClient, err = elasticsearch.NewClient(d.logs[file].config.version, d.logs[file].config.url, d.logs[file].config.username, d.logs[file].config.password, d.logs[file].config.timeout, d.logs[file].config.sniff, d.logs[file].config.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	d.logs[file].pipeline.group, d.logs[file].pipeline.ctx = errgroup.WithContext(ctx)
	d.logs[file].pipeline.inputCh = make(chan logdriver.LogEntry)
	d.logs[file].pipeline.outputCh = make(chan LogMessage)

	// l.Printf("INFO starting: %#v\n", d.logs[file].info.ContainerID)

	d.logs[file].pipeline.group.Go(func() error {

		dec := protoio.NewUint32DelimitedReader(d.logs[file].stream, binary.BigEndian, 1e6)
		defer func() {
			// fmt.Printf("info: [%v] closing dec.\n", d.logs[file].info.ContainerID)
			dec.Close()
		}()
		// defer d.pipeline.ctx.Done()

		var buf logdriver.LogEntry
		var err error

		for {
			if err = dec.ReadMsg(&buf); err != nil {
				if err == io.EOF {
					// fmt.Printf("info: [%v] shutting down log logger: %v\n", d.logs[file].info.ContainerID, err)
					d.logs[file].stream.Close()
					return nil
				}
				if err != nil {
					// l.Printf("error panicing: %v\n", err)
					return err
				}

				dec = protoio.NewUint32DelimitedReader(d.logs[file].stream, binary.BigEndian, 1e6)
			}

			// l.Printf("INFO pipe1 client: %#v\n", d.logs[file].esClient)
			// l.Printf("INFO pipe1 line: %#v\n", string(buf.Line))

			// BUG: (17.09.0~ce-0~debian) docker run command throws lots empty line messages
			if len(bytes.TrimSpace(buf.Line)) == 0 {

				l.Printf("error trimming")
				// TODO: add log debug level
				continue
			}

			select {
			case d.logs[file].pipeline.inputCh <- buf:
			case <-d.logs[file].pipeline.ctx.Done():
				// l.Printf("ERROR pipe1: %#v\n", d.logs[file].pipeline.ctx.Err())
				return d.logs[file].pipeline.ctx.Err()
				// default:
			}
			buf.Reset()
		}
	})

	d.logs[file].pipeline.group.Go(func() error {

		var logMessage string

		// custom log message fields
		msg := getLostashFields(d.logs[file].config.fields, d.logs[file].info)

		groker, err := grok.NewGrok(d.logs[file].config.grokMatch, d.logs[file].config.grokPattern, d.logs[file].config.grokPatternFrom, d.logs[file].config.grokPatternSplitter, d.logs[file].config.grokNamedCapture)
		if err != nil {
			return err
		}

		for m := range d.logs[file].pipeline.inputCh {

			logMessage = string(m.Line)

			// create message
			msg.Source = m.Source
			msg.Partial = m.Partial
			msg.TimeNano = m.TimeNano

			// l.Printf("INFO pipe2: %#v\n", string(m.Line))

			// TODO: create a PR to grok upstream for parsing bytes
			// so that we avoid having to convert the message to string
			msg.GrokLine, msg.Line, err = groker.ParseLine(d.logs[file].config.grokMatch, logMessage, m.Line)
			if err != nil {
				l.Printf("error: [%v] parsing log message: %v\n", d.logs[file].info.ID(), err)
			}

			select {
			case d.logs[file].pipeline.outputCh <- msg:
			case <-d.logs[file].pipeline.ctx.Done():
				// l.Printf("ERROR pipe2: %#v\n", d.logs[file].pipeline.ctx.Err())
				return d.logs[file].pipeline.ctx.Err()
			}

		}

		return nil
	})

	d.logs[file].pipeline.group.Go(func() error {

		err := d.logs[file].esClient.NewBulkProcessorService(d.logs[file].pipeline.ctx, d.logs[file].config.Bulk.workers, d.logs[file].config.Bulk.actions, d.logs[file].config.Bulk.size, d.logs[file].config.Bulk.flushInterval, d.logs[file].config.Bulk.stats)
		if err != nil {
			l.Printf("error creating bulk processor: %v\n", err)
		}

		for {
			// l.Printf("INFO pipe3 starts")
			select {
			case doc := <-d.logs[file].pipeline.outputCh:
				// l.Printf("INFO pipe3: %#v\n", string(doc.Line))
				// l.Printf("sending doc: %#v\n", doc.GrokLine)
				d.logs[file].esClient.Add(d.logs[file].config.index, d.logs[file].config.tzpe, doc)

			case <-d.logs[file].pipeline.ctx.Done():
				// l.Printf("ERROR pipe3: %#v\n", d.logs[file].pipeline.ctx.Err())
				return d.logs[file].pipeline.ctx.Err()
			}
		}

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

	// log.Infof("info: stopping log: %s\n", file)

	// TODO: count how many docs are in the queue before shutting down
	// alternative: sleep flush interval time
	// time.Sleep(10 * time.Second)

	d.mu.Lock()
	c, exists := d.logs[file]
	if !exists {
		return fmt.Errorf("error: logger not found for %v", file)
	}
	d.mu.Unlock()

	if c.stream != nil {
		// l.Printf("error: [%v] closing container stream", c.info.ID())
		c.stream.Close()
	}

	time.Sleep(10 * time.Second)

	if c.esClient != nil {
		// l.Printf("INFO client: %v", c.esClient)
		if err := c.esClient.Close(); err != nil {
			l.Printf("error: closing client connection: %v", err)
		}
		c.esClient.Stop()
	}

	delete(d.logs, file)

	if c.pipeline.group != nil {
		// l.Printf("INFO [%v] closing pipeline: %v", c.info.ContainerID, c.pipeline)

		close(c.pipeline.inputCh)
		close(c.pipeline.outputCh)
		// d.pipeline.group.Stop()

		// Check whether any goroutines failed.
		if err := c.pipeline.group.Wait(); err != nil {
			l.Printf("error with pipeline: %v", err)
		}
	}

	// l.Printf("INFO done stop logging")

	return nil
}

// Name ...
func (d Driver) Name() string {
	return name
}
