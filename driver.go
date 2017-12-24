package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/jsonfilelog"
	protoio "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"

	elastic "gopkg.in/olivere/elastic.v5"
)

const (
	name = "elasticsearch"

	defaultEsHost  = "127.0.0.1"
	defaultEsPort  = 9200
	defaultEsIndex = "docker"
	defaultEsType  = "logs"

	// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-date-format.html
	dateHourMinuteSecondFraction = "2006-01-02T15:04:05.000Z"
	basicOrdinalDateTime         = "20060102T150405Z"
)

type elasticSearch struct {
	Version   string `json:"@version"`
	Host      string `json:"hostname"`
	Log       string `json:"message"`
	Timestamp string `json:"@timestamp"`
	Name      string `json:"name"`
	//Stream string stderr stdout
	ImageID string
}

type loggerContext struct {
	Hostname      string
	ContainerID   string
	ContainerName string
	ImageID       string
	ImageName     string
	Command       string
	Created       time.Time
}

type driver struct {
	mu     sync.Mutex
	logs   map[string]*logPair
	idx    map[string]*logPair
	logger logger.Logger

	esClient       *elastic.Client
	esIndexService *elastic.IndexService
	esCtx          context.Context

	tag string
}

type logPair struct {
	l      logger.Logger
	stream io.ReadCloser
	info   logger.Info
}

func newDriver() *driver {
	return &driver{
		logs: make(map[string]*logPair),
		idx:  make(map[string]*logPair),
	}
}

func (d *driver) StartLogging(file string, logCtx logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	if logCtx.LogPath == "" {
		logCtx.LogPath = filepath.Join("/var/log/docker", logCtx.ContainerID)
	}
	if err := os.MkdirAll(filepath.Dir(logCtx.LogPath), 0755); err != nil {
		return errors.Wrap(err, "error setting up logger dir")
	}
	l, err := jsonfilelog.New(logCtx)
	if err != nil {
		return errors.Wrap(err, "error creating jsonfile logger")
	}

	logrus.WithField("id", logCtx.ContainerID).WithField("file", file).WithField("logpath", logCtx.LogPath).Debugf("Start logging")
	f, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	d.mu.Lock()
	lf := &logPair{l, f, logCtx}
	d.logs[file] = lf
	d.idx[logCtx.ContainerID] = lf
	d.mu.Unlock()

	proto, host, port, err := parseAddress(logCtx.Config["elasticsearch-address"])
	if err != nil {
		return err
	}

	var esClient *elastic.Client
	esClient, err = elastic.NewClient(
		elastic.SetURL(proto + "://" + host + ":" + port),
		//elastic.SetMaxRetries(t.maxRetries),
		//elastic.SetSniff(t.sniff),
		//elastic.SetSnifferInterval(t.snifferInterval),
		//elastic.SetHealthcheck(t.healthcheck),
		//elastic.SetHealthcheckInterval(t.healthcheckInterval))
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: cannot connect to the endpoint: %s://%s:%s\n%v",
			proto,
			host,
			port,
			err,
		)
	}

	esIndex := getCtxConfig(logCtx.Config["elasticsearch-index"], defaultEsIndex)
	esType := getCtxConfig(logCtx.Config["elasticsearch-type"], defaultEsType)

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

type LogMessage struct {
	logger.Message
	logger.Info
	LogLine string
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

func (d *driver) ReadLogs(info logger.Info, config logger.ReadConfig) (io.ReadCloser, error) {
	d.mu.Lock()
	lf, exists := d.idx[info.ContainerID]
	d.mu.Unlock()
	if !exists {
		return nil, fmt.Errorf("logger does not exist for %s", info.ContainerID)
	}

	r, w := io.Pipe()
	lr, ok := lf.l.(logger.LogReader)
	if !ok {
		return nil, fmt.Errorf("logger does not support reading")
	}

	go func() {
		watcher := lr.ReadLogs(config)

		enc := protoio.NewUint32DelimitedWriter(w, binary.BigEndian)
		defer enc.Close()
		defer watcher.Close()

		var buf logdriver.LogEntry
		for {
			select {
			case msg, ok := <-watcher.Msg:
				if !ok {
					w.Close()
					return
				}

				buf.Line = msg.Line
				buf.Partial = msg.Partial
				buf.TimeNano = msg.Timestamp.UnixNano()
				buf.Source = msg.Source

				if err := enc.WriteMsg(&buf); err != nil {
					w.CloseWithError(err)
					return
				}
			case err := <-watcher.Err:
				w.CloseWithError(err)
				return
			}

			buf.Reset()
		}
	}()

	return r, nil
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
		case "elasticsearch-username":
		case "elasticsearch-password":
		case "max-retry":
		case "timeout":
		case "tag":
		case "labels":
		case "env":
		default:
			return fmt.Errorf("unknown log opt %q for elasticsearch log driver", key)
		}
	}

	if _, _, _, err := parseAddress(cfg["elasticsearch-address"]); err != nil {
		return err
	}

	return nil
}
