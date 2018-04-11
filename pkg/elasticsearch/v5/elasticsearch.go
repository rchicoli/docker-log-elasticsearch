package v5

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context"
	"gopkg.in/olivere/elastic.v5"
)

// Elasticsearch ...
type Elasticsearch struct {
	*elastic.Client
	*elastic.BulkProcessor
	*elastic.BulkProcessorService
}

// NewClient ...
func NewClient(address, username, password string, timeout int, sniff bool, insecure bool) (*Elasticsearch, error) {

	url, _ := url.Parse(address)
	tr := new(http.Transport)

	if url.Scheme == "https" {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
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
		Client:               c,
		BulkProcessorService: c.BulkProcessor(),
	}, nil
}

// Log sends log messages to elasticsearch
func (e *Elasticsearch) Log(ctx context.Context, index, tzpe string, msg interface{}) error {
	if _, err := e.Client.Index().Index(index).Type(tzpe).BodyJson(msg).Do(ctx); err != nil {
		return err
	}
	return nil
}

func (e *Elasticsearch) NewBulkProcessorService(ctx context.Context, workers, actions, size int, flushInterval time.Duration, stats bool) error {

	p, err := e.BulkProcessorService.
		Workers(workers).
		BulkActions(actions).         // commit if # requests >= BulkSize
		BulkSize(size).               // commit if size of requests >= 1 MB
		FlushInterval(flushInterval). // commit every given interval
		Stats(stats).                 // collect stats
		// Backoff(backoff).
		Do(ctx)
	if err != nil {
		return err
	}

	e.BulkProcessor = p

	return nil
}

func (e *Elasticsearch) Add(index, tzpe string, msg interface{}) error {

	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg)
	e.BulkProcessor.Add(r)

	return nil
}

func (e *Elasticsearch) Close() error {
	return e.BulkProcessor.Close()
}

func (e *Elasticsearch) Flush() error {
	return e.BulkProcessor.Flush()
}

// Stop stops the background processes that the client is running,
// i.e. sniffing the cluster periodically and running health checks
// on the nodes.
func (e *Elasticsearch) Stop() {
	e.Client.Stop()
}
