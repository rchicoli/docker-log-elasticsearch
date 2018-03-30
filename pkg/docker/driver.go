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
	"regexp"
	"strings"
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

var javaExceptionPattern = regexp.MustCompile("(^.+Exception: .+)|(^\\s+at .+)|(^\\s+... \\d+ more)|(^\\s*Caused by:.+)")

// var clojureExceptionPattern = regexp.MustCompile("^.+#error {|^\\s+:cause .+|^\\s+:via$|:type |:message |:at |^\\s+:trace$|^\\s+\\[?\\[.+\\]\\]?}?$")

var l = log.New(os.Stderr, "", 0)

// Driver ...
type Driver struct {
	mu     *sync.Mutex
	logs   map[string]*container
	logger logger.Logger

	esClient elasticsearch.Client

	groker *grok.Grok
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
		logs: make(map[string]*container),
		mu:   new(sync.Mutex),
	}
}

// StartLogging ...
func (d Driver) StartLogging(file string, info logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("error: logger for %q already exists", file)

	}
	d.mu.Unlock()

	ctx := context.Background()

	// log.Printf("info: starting log: %s\n", file)

	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return fmt.Errorf("error: opening logger fifo: %q", file)
	}

	d.mu.Lock()
	c := &container{stream: f, info: info}
	d.logs[file] = c
	d.mu.Unlock()

	cfg := defaultLogOpt()
	if err := cfg.validateLogOpt(info.Config); err != nil {
		return fmt.Errorf("error: validating log options: %v", err)
	}

	d.esClient, err = elasticsearch.NewClient(cfg.version, cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	d.groker, err = grok.NewGrok(cfg.grokMatch, cfg.grokPattern, cfg.grokPatternFrom, cfg.grokPatternSplitter, cfg.grokNamedCapture)
	if err != nil {
		return err
	}

	msgCh := make(chan LogMessage)
	lineCh := make(chan logdriver.LogEntry)
	logCh := make(chan logdriver.LogEntry)

	g, ectx := errgroup.WithContext(ctx)

	g.Go(func() error {
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
			case logCh <- buf:
			case <-ectx.Done():
				return ectx.Err()
			}
			buf.Reset()
		}
	})

	g.Go(func() error {

		var test logdriver.LogEntry
		multiline := make([]byte, 0)

		for m := range logCh {

			// BUG: (17.09.0~ce-0~debian) docker run command throws lots empty line messages
			// TODO: profile: check for resource consumption
			if len(bytes.TrimSpace(m.Line)) == 0 {
				// TODO: add log debug level
				continue
			}

			matched := javaExceptionPattern.Match(m.Line)
			if matched {
				l.Printf("java EXCEPTION: %v", string(m.Line))
				multiline = append(multiline, m.Line...)
				test = m
				test.Line = multiline
				continue
			} else {

				select {
				case lineCh <- test:
				case lineCh <- m:
				case <-ectx.Done():
					return ectx.Err()
				}

			}

		}

		return nil
	})

	g.Go(func() error {

		var logMessage string

		// custom log message fields
		msg := getLostashFields(cfg.fields, c.info)

		for m := range lineCh {

			logMessage = string(m.Line)

			// BUG: (17.09.0~ce-0~debian) docker run command throws lots empty line messages
			// TODO: profile: check for resource consumption
			if len(strings.TrimSpace(logMessage)) == 0 {
				// TODO: add log debug level
				continue
			}
			// create message
			msg.Source = m.Source
			msg.Partial = m.Partial
			msg.TimeNano = m.TimeNano

			msg.GrokLine, msg.Line, err = d.groker.ParseLine(cfg.grokMatch, logMessage, m.Line)
			if err != nil {
				l.Printf("error: [%v] parsing log message: %v\n", c.info.ID(), err)
			}

			select {
			case msgCh <- msg:
			case <-ectx.Done():
				return ectx.Err()
			}
		}

		return nil
	})

	g.Go(func() error {

		err := d.esClient.NewBulkProcessorService(ectx, cfg.Bulk.workers, cfg.Bulk.actions, cfg.Bulk.size, cfg.Bulk.flushInterval, cfg.Bulk.stats)
		if err != nil {
			l.Printf("error creating bulk processor: %v", err)
		}

		for {
			select {
			case doc := <-msgCh:
				d.esClient.Add(cfg.index, cfg.tzpe, doc)
			case <-ectx.Done():
				return ectx.Err()
			}
		}
	})

	// TODO: create metrics from stats
	// g.Go(func() error {
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

	// Check whether any goroutines failed.
	// if err := g.Wait(); err != nil {
	// 	panic(err)
	// }

	return nil
}

// StopLogging ...
func (d Driver) StopLogging(file string) error {

	// log.Infof("info: stopping log: %s\n", file)

	d.mu.Lock()
	c, ok := d.logs[file]
	if ok {
		c.stream.Close()
		delete(d.logs, file)
	}
	d.mu.Unlock()

	if d.esClient != nil {
		d.esClient.Close()
		d.esClient.Stop()
	}

	return nil
}

// Name ...
func (d Driver) Name() string {
	return name
}
