package v2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"gopkg.in/olivere/elastic.v2"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
)

type Elasticsearch struct {
	Client       *elastic.Client
	indexService *elastic.IndexService
}

func NewClient(address, username, password string, timeout int, sniff bool) (elasticsearch.Client, error) {

	url, _ := url.Parse(address)
	tr := new(http.Transport)

	if url.Scheme == "https" {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	client := &http.Client{Transport: tr}

	c, err := elastic.NewClient(
		elastic.SetURL(address),
		elastic.SetScheme(url.Scheme),
		elastic.SetBasicAuth(username, password),
		elastic.SetHttpClient(client),
		elastic.SetSniff(sniff),
		elastic.SetRetrier(NewMyRetrier(timeout)),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: cannot connect to the endpoint: %s\n%v", url, err)
	}
	return &Elasticsearch{
		Client:       c,
		indexService: c.Index(),
	}, nil
}

// Log sends log messages to elasticsearch
func (e *Elasticsearch) Log(ctx context.Context, index, tzpe string, msg interface{}) error {
	if _, err := e.indexService.Index(index).Type(tzpe).BodyJson(msg).Do(); err != nil {
		return err
	}
	return nil
}
