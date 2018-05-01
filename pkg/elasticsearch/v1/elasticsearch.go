package v1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"gopkg.in/olivere/elastic.v2"
	"gopkg.in/olivere/elastic.v2/backoff"
)

const version = 1

// Elasticsearch client
type Elasticsearch struct {
	*elastic.Client
}

// NewClient creates a new elasticsearch client
func NewClient(address, username, password string, timeout time.Duration, sniff bool, insecure bool) (*Elasticsearch, error) {

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
		Client: c,
	}, nil
}

// Log sends log messages to elasticsearch
func (e *Elasticsearch) Log(_ context.Context, index, tzpe string, msg interface{}) error {
	if _, err := e.Client.Index().Index(index).Type(tzpe).BodyJson(msg).Do(); err != nil {
		return err
	}
	return nil
}

// Version reports the client version
func (e *Elasticsearch) Version() int {
	return version
}

// BulkService ...
type BulkService struct {
	bulkService    *elastic.BulkService
	initialTimeout time.Duration
	timeout        time.Duration
}

// Bulk creates a service
func Bulk(client *Elasticsearch, timeout time.Duration) *BulkService {
	return &BulkService{
		bulkService:    elastic.NewBulkService(client.Client),
		timeout:        timeout,
		initialTimeout: 100 * time.Millisecond,
	}
}

// Add adds bulkable requests, i.e. BulkIndexRequest, BulkUpdateRequest,
// and/or BulkDeleteRequest.
func (e BulkService) Add(index, tzpe string, msg interface{}) {
	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg)
	e.bulkService.Add(r)
}

// CommitRequired returns true if the service has to commit its
// bulk requests. This can be either because the number of actions
// or the estimated size in bytes is larger than specified in the
// BulkProcessorService.
func (e BulkService) CommitRequired(actions int, bulkSize int) bool {
	if actions >= 0 && e.bulkService.NumberOfActions() >= actions {
		return true
	}
	if bulkSize >= 0 && e.bulkService.EstimatedSizeInBytes() >= int64(bulkSize) {
		return true
	}
	return false
}

// Do sends the batched requests to Elasticsearch. Note that, when successful,
// you can reuse the BulkService for the next batch as the list of bulk
// requests is cleared on success.
// {
//   "took":3,
//   "errors":false,
//   "items":[{
//     "index":{
//       "_index":"index1",
//       "_type":"tweet",
//       "_id":"1",
//       "_version":3,
//       "status":201
//     }
//   }
// }
func (e BulkService) Do(context.Context) (interface{}, int, bool, error) {

	var bulkResponse *elastic.BulkResponse

	// commitFunc will commit bulk requests and, on failure, be retried
	// via exponential backoff
	commitFunc := func() error {
		var err error
		bulkResponse, err = e.bulkService.Do()
		return err
	}
	// notifyFunc will be called if retry fails
	notifyFunc := func(_ error, _ time.Duration) {
		// log.Errorf("elastic: bulk processor failed but may retry: %v", err)
	}

	policy := backoff.NewExponentialBackoff(e.initialTimeout, e.timeout).SendStop(true)
	err := backoff.RetryNotify(commitFunc, policy, notifyFunc)

	if bulkResponse != nil {
		return bulkResponse, bulkResponse.Took, bulkResponse.Errors, err
	}
	return nil, 0, true, err

}

// Errors parses a BulkResponse and returns the reason of the failure requests
// {
// 	"error" : {
// 	  "root_cause" : [
// 		{
// 		  "type" : "illegal_argument_exception",
// 		  "reason" : "Failed to parse int parameter [size] with value [surprise_me]"
// 		}
// 	  ],
// 	  "type" : "illegal_argument_exception",
// 	  "reason" : "Failed to parse int parameter [size] with value [surprise_me]",
// 	  "caused_by" : {
// 		"type" : "number_format_exception",
// 		"reason" : "For input string: \"surprise_me\""
// 	  }
// 	},
// 	"status" : 400
//   }
func (e BulkService) Errors(bulkResponse interface{}) []map[int]string {

	if bulkResponse == nil {
		return nil
	}
	if bulkResponse.(*elastic.BulkResponse).Items == nil {
		return nil
	}

	var reason []map[int]string
	for _, item := range bulkResponse.(*elastic.BulkResponse).Items {
		for _, result := range item {
			if result.Error == "" {
				continue
			}
			reason = append(reason, map[int]string{
				result.Status: result.Error,
			})
		}
	}
	return reason

}

// EstimatedSizeInBytes returns the estimated size of all bulkable
// requests added via Add.
func (e BulkService) EstimatedSizeInBytes() int64 {
	return e.bulkService.EstimatedSizeInBytes()
}

// NumberOfActions returns the number of bulkable requests that need to
// be sent to Elasticsearch on the next batch.
func (e BulkService) NumberOfActions() int {
	return e.bulkService.NumberOfActions()
}

// Stop stops the background processes that the client is running,
// i.e. sniffing the cluster periodically and running health checks
// on the nodes.
func (e *Elasticsearch) Stop() {
	e.Client.Stop()
}
