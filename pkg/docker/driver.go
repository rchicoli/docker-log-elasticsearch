package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fifo"
	"github.com/vjeantet/grok"

	"github.com/rchicoli/cfssl/log"
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

type Driver struct {
	mu     sync.Mutex
	logs   map[string]*container
	logger logger.Logger

	esClient elasticsearch.Client

	groker *grok.Grok
}

type container struct {
	stream io.ReadCloser
	info   logger.Info
}

type LogMessage struct {
	logdriver.LogEntry
	logger.Info

	GrokLine map[string]string
}

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

			Line:     strings.TrimSpace(string(l.Line)),
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

func NewDriver() Driver {
	return Driver{
		logs: make(map[string]*container),
	}
}

func (d Driver) StartLogging(file string, info logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	ctx := context.Background()

	logrus.WithField("id", info.ContainerID).WithField("file", file).WithField("logpath", info.LogPath).Debugf("Start logging")
	f, err := fifo.OpenFifo(ctx, file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	d.mu.Lock()
	c := &container{
		stream: f,
		info:   info,
	}
	d.logs[file] = c
	d.mu.Unlock()

	cfg := defaultLogOpt()
	if err := cfg.validateLogOpt(info.Config); err != nil {
		return errors.Wrapf(err, "error: elasticsearch-options: %q", err)
	}
	logrus.WithField("id", info.ContainerID).Debugf("log-opt: %v", cfg)

	switch cfg.version {
	case "1":
		d.esClient, err = elasticv2.NewClient(cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
		if err != nil {
			return fmt.Errorf("elasticsearch: cannot create a client: %v", err)
		}
	case "2":
		d.esClient, err = elasticv3.NewClient(cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
		if err != nil {
			return fmt.Errorf("elasticsearch: cannot create a client: %v", err)
		}
	case "5":
		d.esClient, err = elasticv5.NewClient(cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
		if err != nil {
			return fmt.Errorf("elasticsearch: cannot create a client: %v", err)
		}
	case "6":
		d.esClient, err = elasticv6.NewClient(cfg.url, cfg.username, cfg.password, cfg.timeout, cfg.sniff, cfg.insecure)
		if err != nil {
			return fmt.Errorf("elasticsearch: cannot create a client: %v", err)
		}
	}

	go d.consumeLog(ctx, cfg.tzpe, cfg.index, c, cfg.fields, cfg.grokPattern)
	return nil
}

func (d Driver) consumeLog(ctx context.Context, esType, esIndex string, c *container, fields, grokPattern string) {

	dec := protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
	defer dec.Close()

	if grokPattern != "" {
		d.groker, _ = grok.NewWithConfig(&grok.Config{})
	}

	// custom log message fields
	msg := getLostashFields(fields, c.info)

	var buf logdriver.LogEntry
	var err error

	for {
		if err = dec.ReadMsg(&buf); err != nil {
			if err == io.EOF {
				logrus.WithField("id", c.info.ContainerID).WithError(err).Debug("shutting down log logger")
				c.stream.Close()
				return
			}
			dec = protoio.NewUint32DelimitedReader(c.stream, binary.BigEndian, 1e6)
		}

		// create message
		msg.Source = buf.Source
		msg.Partial = buf.Partial
		msg.GrokLine, msg.Line, err = d.parseLine(grokPattern, buf.Line)
		if err != nil {
			log.Errorf("elasticsearch: grok failed to parse line: %v", err)
		}
		msg.TimeNano = buf.TimeNano

		if err = d.esClient.Log(ctx, esIndex, esType, msg); err != nil {
			logrus.WithField("id", c.info.ContainerID).
				WithError(err).
				WithField("message", msg).
				WithField("line", string(msg.Line)).
				Error("error writing log message")
			continue
		}

		buf.Reset()
	}
}

func (d Driver) parseLine(pattern string, line []byte) (map[string]string, []byte, error) {

	if d.groker == nil {
		return nil, line, nil
	}

	// TODO: create a PR to grok upstream for returning a regexp
	// doing so we avoid to compile the regexp twice
	// TODO: profile line below and perhaps place variables outside this function
	grokMatch, err := d.groker.Match(pattern, string(line))
	if err != nil {
		return nil, nil, err
	}
	if !grokMatch {
		// do not try parse this line, because it will return an empty map
		return map[string]string{"failed": string(line)}, nil, fmt.Errorf("elasticsearch: grok pattern does not match line: %s", string(line))
	}

	grokLine, err := d.groker.Parse(pattern, string(line))
	if err != nil {
		return nil, nil, err
	}

	return grokLine, nil, nil

}

func (d Driver) StopLogging(file string) error {
	logrus.WithField("file", file).Debugf("Stop logging")
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

func (d Driver) Name() string {
	return name
}

func logError(msg interface{}, str string, err error) {
	logrus.WithFields(
		logrus.Fields{
			"message": msg,
			"error":   err,
		},
	).Error(str)
}
