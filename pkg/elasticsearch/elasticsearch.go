package elasticsearch

import (
	"context"
	"fmt"

	elastic "gopkg.in/olivere/elastic.v5"
)

type Elasticsearch struct {
	Client       *elastic.Client
	indexService *elastic.IndexService
}

func NewClient(url string, timeout int) (*Elasticsearch, error) {
	c, err := elastic.NewClient(
		elastic.SetURL(url),
		// elastic.SetMaxRetries(t.maxRetries),
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
	if _, err := e.indexService.Index(index).Type(tzpe).BodyJson(msg).Do(ctx); err != nil {
		return err
	}
	return nil
}
