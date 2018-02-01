package v3

import (
	"context"
	"fmt"

	"gopkg.in/olivere/elastic.v3"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
)

type Elasticsearch struct {
	Client       *elastic.Client
	indexService *elastic.IndexService
}

func NewClient(url string, timeout int) (elasticsearch.Client, error) {
	c, err := elastic.NewClient(
		elastic.SetURL(url),
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
