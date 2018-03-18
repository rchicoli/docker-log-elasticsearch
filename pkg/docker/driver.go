package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/tonistiigi/fifo"
	"github.com/vjeantet/grok"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"

	elasticv2 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v1"
	elasticv3 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v2"
	elasticv5 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v5"
	elasticv6 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v6"

	protoio "github.com/gogo/protobuf/io"
)

const (
	name = "elasticsearchlog"
)

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

	d.esClient, err = NewClient(cfg.version, cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
	if err != nil {
		return fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
	}

	d.groker, err = NewGrok(cfg.grokMatch, cfg.grokPattern, cfg.grokPatternFrom, cfg.grokPatternSplitter, cfg.grokNamedCapture)
	if err != nil {
		return err
	}

	go d.consumeLog(ctx, cfg.tzpe, cfg.index, c, cfg.fields, cfg.grokMatch)

	return nil
}

func NewGrok(grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter string, grokNamedCapture bool) (*grok.Grok, error) {
	if grokMatch == "" {
		return nil, nil
	}

	groker, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: grokNamedCapture})

	if grokPattern != "" {
		var patternNames []string
		grokPatterns := strings.Split(grokPattern, grokPatternSplitter)
		for _, v := range grokPatterns {
			patternNames = strings.Split(v, "=")
			if len(patternNames) != 2 {
				return groker, fmt.Errorf("error: parsing grok-pattern, missing '=' separator")
			}
			err := groker.AddPattern(patternNames[0], patternNames[1])
			if err != nil {
				return groker, fmt.Errorf("error: adding grok pattern: %v", err)
			}
		}
	}

	if grokPatternFrom != "" {
		err := groker.AddPatternsFromPath(grokPatternFrom)
		if err != nil {
			return groker, fmt.Errorf("error: adding grok pattern from %s: %v", grokPatternFrom, err)
		}
	}

	return groker, nil
}

func (d Driver) consumeLog(ctx context.Context, esType, esIndex string, c *container, fields, grokMatch string) {

	dec := protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
	defer dec.Close()

	// custom log message fields
	msg := getLostashFields(fields, c.info)

	var buf logdriver.LogEntry
	var err error
	var logMessage string

	for {
		if err = dec.ReadMsg(&buf); err != nil {
			if err == io.EOF {
				// log.Infof("info: [%v] shutting down log logger: %v", c.info.ContainerID, err)
				c.stream.Close()
				return
			}
			dec = protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
		}

		logMessage = string(buf.Line)

		// BUG(17.09.0~ce-0~debian): docker run throws lots empty line messages
		// TODO: profile: check for resource consumption
		if len(strings.TrimSpace(logMessage)) == 0 {
			// TODO: add log debug level
			continue
		}

		// create message
		msg.Source = buf.Source
		msg.Partial = buf.Partial
		msg.GrokLine, msg.Line, err = d.parseLine(grokMatch, logMessage, buf.Line)
		if err != nil {
			l.Printf("error: [%v] parsing log message: %v\n", c.info.ID(), err)
		}
		msg.TimeNano = buf.TimeNano

		if err = d.esClient.Log(ctx, esIndex, esType, msg); err != nil {
			l.Printf("error: [%v] writing log message: %v\n", c.info.ID(), err)
			continue
		}

		buf.Reset()
	}
}

// NewClient ...
func NewClient(version string, url, username, password string, timeout int, sniff bool, insecure bool) (elasticsearch.Client, error) {
	switch version {
	case "1":
		client, err := elasticv2.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "2":
		client, err := elasticv3.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "5":
		client, err := elasticv5.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "6":
		client, err := elasticv6.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("error: elasticsearch version not supported: %v", version)
	}
}

func (d Driver) parseLine(pattern, logMessage string, line []byte) (map[string]string, []byte, error) {

	if d.groker == nil {
		return nil, line, nil
	}

	// TODO: create a PR to grok upstream for returning a regexp
	// doing so we avoid to compile the regexp twice
	// TODO: profile line below and perhaps place variables outside this function
	grokMatch, err := d.groker.Match(pattern, logMessage)
	if err != nil {
		return map[string]string{"line": logMessage, "err": err.Error()}, nil, err
	}
	if !grokMatch {
		// do not try parse this line, because it will return an empty map
		return map[string]string{"line": logMessage, "err": "grok pattern does not match log line"},
			nil,
			fmt.Errorf("error: grok pattern does not match line: %s", logMessage)
	}

	grokLine, err := d.groker.Parse(pattern, logMessage)
	if err != nil {
		return map[string]string{"line": logMessage, "err": err.Error()}, nil, err
	}

	return grokLine, nil, nil

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
		d.esClient.Stop()
	}

	return nil
}

// Name ...
func (d Driver) Name() string {
	return name
}
