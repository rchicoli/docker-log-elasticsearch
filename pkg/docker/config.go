package docker

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

type LogOpt struct {
	index   string
	tzpe    string
	url     string
	timeout int
}

func defaultLogOpt() *LogOpt {
	return &LogOpt{
		// TODO: update index name to docker-YYYY.MM.dd
		index:   "docker",
		tzpe:    "log",
		timeout: 1,
	}
}

func parseAddress(address string) error {
	if address == "" {
		return nil
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
		return fmt.Errorf("elastic: please provide elasticsearch-address as proto://host:port")
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
