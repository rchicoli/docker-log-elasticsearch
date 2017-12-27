package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/backend"
	"github.com/rchicoli/docker-log-elasticsearch/elasticsearch"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fifo"

	elastic "gopkg.in/olivere/elastic.v5"
)

const (
	name = "elasticsearch"

	defaultEsHost  = "127.0.0.1"
	defaultEsPort  = 9200
	defaultEsIndex = "docker"
	defaultEsType  = "log"

	// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-date-format.html
	dateHourMinuteSecondFraction = "2006-01-02T15:04:05.000Z"
	basicOrdinalDateTime         = "20060102T150405Z"
)

type LoggerInfo struct {
	Config              map[string]string `json:"config"`
	ContainerID         string            `json:"containerID"`
	ContainerName       string            `json:"containerName"`
	ContainerEntrypoint string            `json:"containerEntrypoint"`
	ContainerArgs       []string          `json:"containerArgs"`
	ContainerImageID    string            `json:"containerImageID"`
	ContainerImageName  string            `json:"containerImageName"`
	ContainerCreated    time.Time         `json:"containerCreated"`
	ContainerEnv        []string          `json:"containerEnv"`
	ContainerLabels     map[string]string `json:"containerLabels"`
	LogPath             string            `json:"logPath"`
	DaemonName          string            `json:"daemonName"`
}

type driver struct {
	mu     sync.Mutex
	logs   map[string]*logPair
	logger logger.Logger

	esClient       *elastic.Client
	esIndexService *elastic.IndexService
	esCtx          context.Context

	tag string
}

type logPair struct {
	stream io.ReadCloser
	info   logger.Info
}

type LogMessage struct {
	// logger.Message
	Line      []byte            `json:"-"`
	Source    string            `json:"source"`
	Timestamp time.Time         `json:"@timestamp"`
	Attrs     []backend.LogAttr `json:"attr,omitempty"`
	Partial   bool              `json:"partial"`

	// Err is an error associated with a message. Completeness of a message
	// with Err is not expected, tho it may be partially complete (fields may
	// be missing, gibberish, or nil)
	Err error `json:"err,omitempty"`

	LoggerInfo
	LogLine string `json:"message"`
}

func newDriver() *driver {
	return &driver{
		logs: make(map[string]*logPair),
	}
}

func (d *driver) StartLogging(file string, info logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	logrus.WithField("id", info.ContainerID).WithField("file", file).WithField("logpath", info.LogPath).Debugf("Start logging")
	f, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	d.mu.Lock()
	lf := &logPair{
		stream: f,
		info:   info,
	}
	d.logs[file] = lf
	d.mu.Unlock()

	proto, host, port, err := parseAddress(info.Config["elasticsearch-address"])
	if err != nil {
		return err
	}

	var esClient *elastic.Client
	esClient, err = elastic.NewClient(
		elastic.SetURL(proto+"://"+host+":"+port),
		//elastic.SetMaxRetries(t.maxRetries),
		//elastic.SetSniff(t.sniff),
		//elastic.SetSnifferInterval(t.snifferInterval),
		//elastic.SetHealthcheck(t.healthcheck),
		//elastic.SetHealthcheckInterval(t.healthcheckInterval))
		elastic.SetRetrier(elasticsearch.NewMyRetrier()),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: cannot connect to the endpoint: %s://%s:%s\n%v",
			proto,
			host,
			port,
			err,
		)
	}

	esIndex := getCtxConfig(info.Config["elasticsearch-index"], defaultEsIndex)
	esType := getCtxConfig(info.Config["elasticsearch-type"], defaultEsType)

	d.esClient = esClient
	d.esIndexService = d.esClient.Index()

	esCtx := context.Background()
	d.esCtx = esCtx

	var createIndex *elastic.IndicesCreateResult
	if exists, _ := esClient.IndexExists(esIndex).Do(d.esCtx); !exists {
		createIndex, err = esClient.CreateIndex(esIndex).Do(esCtx)
		if err != nil {
			return fmt.Errorf("elasticsearch: cannot create Index to elasticsearch: %v", err)
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("elasticsearch: index not Acknowledged: %v", err)
		}
	}

	go d.consumeLog(esType, esIndex, lf)
	return nil
}

func (d *driver) consumeLog(esType, esIndex string, lf *logPair) {

	dec := protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
	defer dec.Close()

	var buf logdriver.LogEntry
	var msg LogMessage

	for {
		if err := dec.ReadMsg(&buf); err != nil {
			if err == io.EOF {
				logrus.WithField("id", lf.info.ContainerID).WithError(err).Debug("shutting down log logger")
				lf.stream.Close()
				return
			}
			dec = protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
		}

		// create message
		msg.Timestamp = time.Unix(0, buf.TimeNano)
		msg.Source = buf.Source
		msg.Partial = buf.Partial
		msg.LogLine = string(buf.Line)

		msg.Config = lf.info.Config
		msg.ContainerID = lf.info.ContainerID
		msg.ContainerName = lf.info.ContainerName
		msg.ContainerEntrypoint = lf.info.ContainerEntrypoint
		msg.ContainerArgs = lf.info.ContainerArgs
		msg.ContainerImageID = lf.info.ContainerImageID
		msg.ContainerImageName = lf.info.ContainerImageName
		msg.ContainerCreated = lf.info.ContainerCreated
		msg.ContainerEnv = lf.info.ContainerEnv
		msg.ContainerLabels = lf.info.ContainerLabels
		msg.LogPath = lf.info.LogPath
		msg.DaemonName = lf.info.DaemonName

		if err := d.log(esIndex, esType, msg); err != nil {
			logrus.WithField("id", lf.info.ContainerID).
				WithError(err).
				WithField("message", msg).
				Error("error writing log message")
			continue
		}

		buf.Reset()
	}
}

// log send log messages to elasticsearch
func (d *driver) log(esIndex, esType string, msg LogMessage) error {
	if _, err := d.esIndexService.Index(esIndex).Type(esType).BodyJson(msg).Do(d.esCtx); err != nil {
		return err
	}
	return nil
}

func (d *driver) StopLogging(file string) error {
	logrus.WithField("file", file).Debugf("Stop logging")
	d.mu.Lock()
	lf, ok := d.logs[file]
	if ok {
		lf.stream.Close()
		delete(d.logs, file)
	}
	d.mu.Unlock()
	return nil
}

func parseAddress(address string) (string, string, string, error) {
	if address == "" {
		return "", "", "", nil
	}
	//if !urlutil.IsTransportURL(address) {
	//	return "", fmt.Errorf("es-address should be in form proto://address, got %v", address)
	//}
	url, err := url.Parse(address)
	if err != nil {
		return "", "", "", err
	}

	//if url.Scheme != "http" {
	//	return "", fmt.Errorf("es: endpoint needs to be UDP")
	//}

	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return "", "", "", fmt.Errorf("elastic: please provide elasticsearch-address as proto://host:port")
	}

	return url.Scheme, host, port, nil
}

func getCtxConfig(cfg, dfault string) string {
	if cfg == "" {
		return dfault
	}
	return cfg
}

// ValidateLogOpt looks for es specific log option es-address.
func ValidateLogOpt(cfg map[string]string) error {
	for key := range cfg {
		switch key {
		case "elasticsearch-address":
		case "elasticsearch-index":
		case "elasticsearch-type":
		// case "elasticsearch-username":
		// case "elasticsearch-password":
		case "max-retry":
		case "timeout":
		// case "tag":
		// case "labels":
		// case "env":
		default:
			return fmt.Errorf("unknown log opt %q for elasticsearch log driver", key)
		}
	}

	if _, _, _, err := parseAddress(cfg["elasticsearch-address"]); err != nil {
		return err
	}

	return nil
}
