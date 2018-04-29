package v5

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"gopkg.in/olivere/elastic.v5"
)

// Elasticsearch ...
type Elasticsearch struct {
	*elastic.Client
	// *elastic.BulkProcessor
	// *elastic.BulkProcessorService
	*elastic.BulkService
	*elastic.BulkResponse
}

// type BulkResponse = elastic.BulkResponse

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
		Client: c,
		// BulkProcessorService: c.BulkProcessor(),
		BulkService: c.Bulk(),
	}, nil
}

// Log sends log messages to elasticsearch
func (e *Elasticsearch) Log(ctx context.Context, index, tzpe string, msg interface{}) error {
	if _, err := e.Client.Index().Index(index).Type(tzpe).BodyJson(msg).Do(ctx); err != nil {
		return err
	}
	return nil
}

// Add adds bulkable requests, i.e. BulkIndexRequest, BulkUpdateRequest,
// and/or BulkDeleteRequest.
func (e *Elasticsearch) Add(index, tzpe string, msg interface{}) {
	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg)
	e.BulkService.Add(r)
}

// CommitRequired returns true if the service has to commit its
// bulk requests. This can be either because the number of actions
// or the estimated size in bytes is larger than specified in the
// BulkProcessorService.
func (e *Elasticsearch) CommitRequired(actions int, bulkSize int) bool {
	if e.BulkService.NumberOfActions() >= actions {
		return true
	}
	if e.BulkService.EstimatedSizeInBytes() >= int64(bulkSize) {
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
func (e *Elasticsearch) Do(ctx context.Context) (interface{}, int, bool, error) {
	bulkResponse, err := e.BulkService.Do(ctx)
	return bulkResponse, bulkResponse.Took, bulkResponse.Errors, err
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
func (e *Elasticsearch) Errors(bulkResponse interface{}) []map[int]string {
	var reason []map[int]string
	for _, item := range bulkResponse.(*elastic.BulkResponse).Items {
		for _, result := range item {
			reason = append(reason, map[int]string{
				result.Status: result.Error.Reason,
			})
		}
	}
	return reason
}

// EstimatedSizeInBytes returns the estimated size of all bulkable
// requests added via Add.
func (e *Elasticsearch) EstimatedSizeInBytes() int64 {
	return e.BulkService.EstimatedSizeInBytes()
}

// NumberOfActions returns the number of bulkable requests that need to
// be sent to Elasticsearch on the next batch.
func (e *Elasticsearch) NumberOfActions() int {
	return e.BulkService.NumberOfActions()
}

// Stop stops the background processes that the client is running,
// i.e. sniffing the cluster periodically and running health checks
// on the nodes.
func (e *Elasticsearch) Stop() {
	e.Client.Stop()
}

// func (e *Elasticsearch) NewBulkProcessorService(ctx context.Context, workers, actions, size int, flushInterval time.Duration, stats bool) error {
// 	p, err := e.BulkProcessorService.
// 		Workers(workers).
// 		BulkActions(actions).         // commit if # requests >= BulkSize
// 		BulkSize(size).               // commit if size of requests >= 1 MB
// 		FlushInterval(flushInterval). // commit every given interval
// 		Stats(stats).                 // collect stats
// 		// Backoff(backoff).
// 		Do(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	e.BulkProcessor = p
// 	return nil
// }
// func (e *Elasticsearch) Add(index, tzpe string, msg interface{}) error {
// 	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg)
// 	e.BulkProcessor.Add(r)
// 	return nil
// }
// func (e *Elasticsearch) Close() error {
// 	return e.BulkProcessor.Close()
// }
// func (e *Elasticsearch) Flush() error {
// 	return e.BulkProcessor.Flush()
// }
