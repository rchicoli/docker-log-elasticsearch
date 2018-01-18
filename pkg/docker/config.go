package docker

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/docker/daemon/logger"
	"github.com/pkg/errors"
)

type LogOpt struct {
	index   string
	tzpe    string
	url     string
	timeout int
	fields  string
}

func defaultLogOpt() *LogOpt {
	return &LogOpt{
		// TODO: update index name to docker-YYYY.MM.dd
		index:   "docker",
		tzpe:    "log",
		timeout: 1,
		fields:  "containerID,containerName,containerImageName,containerCreated",
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

	if url.Scheme != "http" {
		return fmt.Errorf("elasticsearch: endpoint accepts only http at the moment")
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
		// case "elasticsearch-username":
		// case "elasticsearch-password":
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
				default:
					return fmt.Errorf("logstash-fields: invalid parameter %s", v)
				}
			}
			c.fields = v
		case "elasticsearch-timeout":
			timeout, err := strconv.Atoi(v)
			if err != nil {
				return errors.Wrapf(err, "error: elasticsearch-timeout: %q", err)
			}
			c.timeout = timeout
		// case "tag":
		// case "labels":
		// case "env":
		default:
			return fmt.Errorf("unknown log opt %q for elasticsearch log Driver", key)
		}
	}

	return nil
}

func getLostashFields(fields string, info logger.Info) LogMessage {
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
	// TODO: omityempty does not work for type time
	l.ContainerCreated = info.ContainerCreated
	return l
}
