package docker

import (
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
	mu        *sync.Mutex
	container *container
	pipeline  pipeline
	esClient  elasticsearch.Client
}

type pipeline struct {
	group    *errgroup.Group
	ctx      context.Context
	outputCh chan LogMessage
	inputCh  chan logdriver.LogEntry
	stopCh   chan struct{}
}

type container struct {
	stream io.ReadCloser
	info   logger.Info
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
func NewDriver() Driver {
	return Driver{
		mu: new(sync.Mutex),
	}
}

// StartLogging ...
func (d *Driver) StartLogging(file string, info logger.Info) error {

	// log.Printf("info: starting log: %s\n", file)

	ctx := context.Background()

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("error: opening logger fifo: %q", file)
	}

	d.mu.Lock()
	c := &container{stream: f, info: info}
	d.container = c
	d.mu.Unlock()

	cfg := defaultLogOpt()
	if err := cfg.validateLogOpt(info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
	}

	d.esClient, err = elasticsearch.NewClient(cfg.version, cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	d.pipeline.group, d.pipeline.ctx = errgroup.WithContext(ctx)
	d.pipeline.inputCh = make(chan logdriver.LogEntry)
	d.pipeline.outputCh = make(chan LogMessage)
	// d.pipeline.stopCh = make(chan struct{})

	d.pipeline.group.Go(func() error {

		dec := protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
		defer dec.Close()

		var buf logdriver.LogEntry
		var err error

		for {
			if err = dec.ReadMsg(&buf); err != nil {
				if err == io.EOF {
					// log.Infof("info: [%v] shutting down log logger: %v", c.info.ContainerID, err)
					c.stream.Close()
					return nil
				}
				dec = protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
			}

			select {
			case d.pipeline.inputCh <- buf:
			// case <-d.pipeline.stopCh:
			// 	return nil
			case <-d.pipeline.ctx.Done():
				return d.pipeline.ctx.Err()
			}
			buf.Reset()
		}
	})

	d.pipeline.group.Go(func() error {

		var logMessage string

		// custom log message fields
		msg := getLostashFields(cfg.fields, c.info)

		groker, err := grok.NewGrok(cfg.grokMatch, cfg.grokPattern, cfg.grokPatternFrom, cfg.grokPatternSplitter, cfg.grokNamedCapture)
		if err != nil {
			return err
		}

		for m := range d.pipeline.inputCh {

			logMessage = string(m.Line)

			// BUG: (17.09.0~ce-0~debian) docker run command throws lots empty line messages
			if len(m.Line) == 0 {
				// TODO: add log debug level
				continue
			}
			// create message
			msg.Source = m.Source
			msg.Partial = m.Partial
			msg.TimeNano = m.TimeNano

			// TODO: create a PR to grok upstream for parsing bytes
			// so that we avoid having to convert the message to string
			msg.GrokLine, msg.Line, err = groker.ParseLine(cfg.grokMatch, logMessage, m.Line)
			if err != nil {
				l.Printf("error: [%v] parsing log message: %v\n", c.info.ID(), err)
			}

			// l.Printf("INFO: grokline: %v\n", msg.GrokLine)
			// l.Printf("INFO: line: %v\n", string(msg.Line))

			select {
			case d.pipeline.outputCh <- msg:
			case <-d.pipeline.ctx.Done():
				return d.pipeline.ctx.Err()
			}

		}

		return nil
	})

	d.pipeline.group.Go(func() error {

		err := d.esClient.NewBulkProcessorService(d.pipeline.ctx, cfg.Bulk.workers, cfg.Bulk.actions, cfg.Bulk.size, cfg.Bulk.flushInterval, cfg.Bulk.stats)
		if err != nil {
			l.Printf("error creating bulk processor: %v\n", err)
		}

		// l.Printf("receving from output\n")

		for doc := range d.pipeline.outputCh {

			// l.Printf("sending doc: %#v\n", doc.GrokLine)
			d.esClient.Add(cfg.index, cfg.tzpe, doc)

			select {
			case <-d.pipeline.ctx.Done():
				return d.pipeline.ctx.Err()
			default:
			}
		}
		// for {
		// 	select {
		// 	case doc := <-d.pipeline.outputCh:
		// 		l.Printf("sending doc: %#v\n", doc.GrokLine)
		// 		d.esClient.Add(cfg.index, cfg.tzpe, doc)

		// 	case <-d.pipeline.ctx.Done():
		// 		return d.pipeline.ctx.Err()
		// 	}
		// }

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

	// log.Infof("info: stopping log: %s\n", file)

	// TODO: count how many docs are in the queue before shutting down
	// alternative: sleep flush interval time
	time.Sleep(10 * time.Second)

	if d.container != nil {
		// l.Printf("INFO container: %v", d.container)
		if err := d.container.stream.Close(); err != nil {
			l.Printf("error: [%v] closing container stream: %v", d.container.info.ID(), err)
		}
	}

	if d.pipeline.group != nil {
		// l.Printf("INFO with pipeline: %v", d.pipeline)

		// close(d.pipeline.inputCh)
		// close(d.pipeline.outputCh)
		d.pipeline.group.Stop()
		// d.pipeline.stopCh <- struct{}{}

		// TODO: close channels gracefully
		// sadly the errgroup does not export cancel function
		// create a PR to golang team with Stop() function
		// Check whether any goroutines failed.
		// if err := d.pipeline.group.Wait(); err != nil {
		// 	l.Printf("error with pipeline: %v", err)
		// }
	}

	if d.esClient != nil {
		// l.Printf("INFO client: %v", d.esClient)
		if err := d.esClient.Close(); err != nil {
			l.Printf("error: closing client connection: %v", err)
		}
		d.esClient.Stop()
	}

	// l.Printf("INFO done stop logging")

	return nil
}

// Name ...
func (d Driver) Name() string {
	return name
}
