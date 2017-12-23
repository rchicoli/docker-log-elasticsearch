// +build linux

// Package elasticsearch provides the log driver for forwarding server logs to
// endpoints that support the Graylog Extended Log Format.
package elasticsearch

import (
	"bytes"

	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/loggerutils"
	elastic "gopkg.in/olivere/elastic.v3"
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

type esLogger struct {
	writer  *elastic.Client
	ctx     loggerContext
	esType  string
	esIndex string
	tag     string
}

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

func init() {
	if err := logger.RegisterLogDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates a elasticsearch logger using the configuration passed in on the
// context. The supported context configuration variable is elasticsearch-address.
func New(ctx logger.Context) (logger.Logger, error) {

	proto, host, port, err := parseAddress(ctx.Config["elasticsearch-address"])
	if err != nil {
		return nil, err
	}

	var writer *elastic.Client
	writer, err = elastic.NewClient(
		elastic.SetURL(proto + "://" + host + ":" + port),
		//elastic.SetMaxRetries(t.maxRetries),
		//elastic.SetSniff(t.sniff),
		//elastic.SetSnifferInterval(t.snifferInterval),
		//elastic.SetHealthcheck(t.healthcheck),
		//elastic.SetHealthcheckInterval(t.healthcheckInterval))
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: cannot connect to the endpoint: %s://%s:%s\n%v",
			proto,
			host,
			port,
			err,
		)
	}

	// remove trailing slash from container name
	containerName := bytes.TrimLeft([]byte(ctx.ContainerName), "/")

	var tag string
	tag, err = loggerutils.ParseLogTag(ctx, "")
	if err != nil {
		return nil, err
	}

	var hostname string
	hostname, err = ctx.Hostname()
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: cannot access hostname to set source field")
	}

	loggerCxt := loggerContext{
		Hostname:      hostname,
		ContainerID:   ctx.ContainerID,
		ContainerName: string(containerName),
		ImageID:       ctx.ContainerImageID,
		ImageName:     ctx.ContainerImageName,
		Command:       ctx.Command(),
		Created:       ctx.ContainerCreated,
	}

	esIndex := getCtxConfig(ctx.Config["elasticsearch-index"], defaultEsIndex)
	esType := getCtxConfig(ctx.Config["elasticsearch-type"], defaultEsType)

	var createIndex *elastic.IndicesCreateResult
	if exists, _ := writer.IndexExists(esIndex).Do(); !exists {
		createIndex, err = writer.CreateIndex(esIndex).Do()
		if err != nil {
			return nil, fmt.Errorf("elasticsearch: cannot create Index to elasticsearch: %v", err)
		}
		if !createIndex.Acknowledged {
			return nil, fmt.Errorf("elasticsearch: index not Acknowledged: %v", err)
		}
	}

	return &esLogger{
		writer:  writer,
		ctx:     loggerCxt,
		esIndex: esIndex,
		esType:  esType,
		tag:     tag,
	}, nil
}

func (s *esLogger) Log(msg *logger.Message) error {
	//level := elastic.LOG_INFO
	//if msg.Source == "stderr" {
	//	level = elastic.LOG_ERR
	//}
	fmt.Printf("CTX: %v", s.ctx)
	fmt.Println()

	m := elasticSearch{
		Version:   "1.0",
		Host:      s.ctx.Hostname,
		Log:       string(msg.Line),
		Timestamp: time.Time(msg.Timestamp).UTC().Format(dateHourMinuteSecondFraction),
		Name:      s.ctx.ContainerName,
		ImageID:   s.ctx.ImageID,
	}

	fmt.Printf("Debugging: Elasticsearch: %v", m)
	fmt.Printf("Debugging: Index: %v", s.esIndex)
	fmt.Printf("Debugging: Type: %v", s.esType)

	_, err := s.writer.Index().
		Index(s.esIndex).
		Type(s.esType).
		BodyJson(m).
		Refresh(true).
		Do()
	if err != nil {
		return fmt.Errorf("elasticsearch: cannot send elastic message: %v", err)
	}

	return nil
}

func (s *esLogger) Close() error {
	return fmt.Errorf("s.writer.Close()")
}

func (s *esLogger) Name() string {
	return name
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
