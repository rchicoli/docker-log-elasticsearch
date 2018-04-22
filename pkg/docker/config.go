package docker

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/daemon/logger"
)

// LogOpt ...
type LogOpt struct {
	index    string
	tzpe     string
	url      string
	timeout  int
	fields   string
	version  string
	username string
	password string
	sniff    bool
	insecure bool

	Bulk

	Grok
}

// Bulk configures the Bulk Processor Service
type Bulk struct {
	workers       int
	actions       int
	size          int
	flushInterval time.Duration
	stats         bool
}

// Grok ...
type Grok struct {
	grokPattern         string
	grokPatternFrom     string
	grokPatternSplitter string
	grokMatch           string
	grokNamedCapture    bool
}

func defaultLogOpt() LogOpt {
	return LogOpt{
		index:    "docker-%Y.%m.%d",
		tzpe:     "log",
		timeout:  1,
		fields:   "containerID,containerName,containerImageName,containerCreated",
		version:  "5",
		sniff:    true,
		insecure: false,

		Bulk: Bulk{
			workers:       1,
			actions:       100,
			size:          5 << 20,
			flushInterval: 5 * time.Second,
			stats:         false,
		},

		Grok: Grok{
			grokPatternSplitter: " and ",
			grokNamedCapture:    true,
		},
	}
}

func parseAddress(address string) error {
	if address == "" {
		return fmt.Errorf("error parsing elasticsearch url")
	}

	url, err := url.Parse(address)
	if err != nil {
		return err
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return fmt.Errorf("elasticsearch: endpoint accepts only http/https, but provided: %v", url.Scheme)
	}

	_, _, err = net.SplitHostPort(url.Host)
	if err != nil {
		return fmt.Errorf("elastic: please provide elasticsearch-url as proto://host:port")
	}

	return nil
}

// ValidateLogOpt looks for es specific log option es-address.
func (c *LogOpt) validateLogOpt(cfg map[string]string) error {
	for key, v := range cfg {
		switch key {
		case "elasticsearch-url":
			if err := parseAddress(v); err != nil {
				return err
			}
			c.url = v
		case "elasticsearch-index":
			c.index = v
		case "elasticsearch-type":
			c.tzpe = v
		case "elasticsearch-username":
			c.username = v
		case "elasticsearch-password":
			c.password = v
		// case "max-retry":
		case "elasticsearch-fields":
			for _, v := range strings.Split(v, ",") {
				switch v {
				case "config":
				case "containerID":
				case "containerName":
				case "containerEntrypoint":
				case "containerArgs":
				case "containerImageID":
				case "containerImageName":
				case "containerCreated":
				case "containerEnv":
				case "containerLabels":
				// case "logPath":
				case "daemonName":
				case "none", "null", "":
				default:
					return fmt.Errorf("error: invalid parameter elasticsearch-fields: %s", v)
				}
			}
			c.fields = v
		case "elasticsearch-sniff":
			s, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-sniff: %q", err)
			}
			c.sniff = s
		case "elasticsearch-insecure":
			s, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-insecure: %q", err)
			}
			c.insecure = s
		case "elasticsearch-version":
			switch v {
			case "1", "2", "5", "6":
				c.version = v
			default:
				return fmt.Errorf("error: elasticsearch-version not supported: %s", v)
			}
		case "elasticsearch-timeout":
			timeout, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-timeout: %q", err)
			}
			c.timeout = timeout

		case "elasticsearch-bulk-workers":
			workers, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-bulk-workers: %q", err)
			}
			c.Bulk.workers = workers
		case "elasticsearch-bulk-actions":
			actions, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-bulk-actions: %q", err)
			}
			c.Bulk.actions = actions
		case "elasticsearch-bulk-size":
			size, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-bulk-size: %q", err)
			}
			c.Bulk.size = size
		case "elasticsearch-bulk-flush-interval":
			flushInterval, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-bulk-flush-interval: %q", err)
			}
			c.Bulk.flushInterval = flushInterval
		case "elasticsearch-bulk-stats":
			stats, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("error: parsing elasticsearch-bulk-stats: %q", err)
			}
			c.Bulk.stats = stats

		case "grok-pattern":
			c.grokPattern = v
		case "grok-pattern-from":
			c.grokPatternFrom = v
		case "grok-pattern-splitter":
			c.grokPatternSplitter = v
		case "grok-match":
			c.grokMatch = v
		case "grok-named-capture":
			s, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("error: parsing grok-named-capture: %q", err)
			}
			c.grokNamedCapture = s

			// case "tag":
		// case "labels":
		// case "env":
		default:
			return fmt.Errorf("error: unknown log-opt: %q", v)
		}
	}

	return nil
}

func getLogOptFields(fields string, info logger.Info) LogMessage {
	var l LogMessage
	for _, v := range strings.Split(fields, ",") {
		switch v {
		case "config":
			l.Config = info.Config
		case "containerID":
			l.ContainerID = info.ID()
		case "containerName":
			l.ContainerName = info.Name()
		case "containerEntrypoint":
			l.ContainerEntrypoint = info.ContainerEntrypoint
		case "containerArgs":
			l.ContainerArgs = info.ContainerArgs
		case "containerImageID":
			l.ContainerImageID = info.ContainerImageID
		case "containerImageName":
			l.ContainerImageName = info.ContainerImageName
		case "containerCreated":
			l.ContainerCreated = info.ContainerCreated
		case "containerEnv":
			l.ContainerEnv = info.ContainerEnv
		case "containerLabels":
			l.ContainerLabels = info.ContainerLabels
		// case "logPath":
		// 	l.LogPath = info.LogPath
		case "daemonName":
			l.DaemonName = info.DaemonName
		default:
		}
	}

	return l
}
