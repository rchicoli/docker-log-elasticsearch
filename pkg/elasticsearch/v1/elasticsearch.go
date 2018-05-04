package v1

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/google/uuid"
	"gopkg.in/olivere/elastic.v2"
	"gopkg.in/olivere/elastic.v2/backoff"
)

const version = 1

// Elasticsearch ...
type Elasticsearch struct {
	*elastic.Client
	*elastic.BulkProcessor
	*elastic.BulkProcessorService
}

// NewClient ...
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
		Client:               c,
		BulkProcessorService: c.BulkProcessor(),
	}, nil
}

// Log sends log messages to elasticsearch
func (e *Elasticsearch) Log(_ context.Context, index, tzpe string, msg interface{}) error {
	if _, err := e.Client.Index().Index(index).Type(tzpe).BodyJson(msg).Do(); err != nil {
		return err
	}
	return nil
}

func (e *Elasticsearch) NewBulkProcessorService(_ context.Context, workers, actions, size int, flushInterval, timeout time.Duration, stats bool, log *logrus.Entry) error {

	afterFunc := func(executionId int64, bulkableRequests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {

		if response != nil && response.Errors {
			// map all requests in order to log the one who's failed
			requests, perr := parseRequest(bulkableRequests)
			if perr != nil {
				log.WithError(err).Error("could not parse request")
			}

			// find out the reasons of the failure
			for _, result := range response.Failed() {
				if result.Error == "" {
					continue
				}
				log.WithFields(logrus.Fields{
					"workerId":  executionId,
					"requestId": result.Id,
					"request":   requests.ById(result.Id),
					"reason":    result.Error,
					"status":    result.Status,
				}).Error("response error message and status code")
			}
		}

		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"workerId": executionId,
				"requests": bulkableRequests,
				"response": response,
			}).Error("after func")
		}
	}

	p, err := e.BulkProcessorService.
		Workers(workers).
		BulkActions(actions).         // commit if # requests >= BulkSize
		BulkSize(size).               // commit if size of requests >= 1 MB
		FlushInterval(flushInterval). // commit every given interval
		Stats(stats).                 // collect stats
		After(afterFunc).
		Do()
	if err != nil {
		return err
	}

	e.BulkProcessor = p

	return nil
}

func (e *Elasticsearch) Add(index, tzpe string, msg interface{}) error {
	id := uuid.New().String()
	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg).Id(id)
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

// Version reports the client version
func (e *Elasticsearch) Version() int {
	return version
}

type mapRequests struct {
	requests map[string]string
}

type Payload struct {
	Index `json:"index"`
}

type Index struct {
	ID    string `json:"_id"`
	Index string `json:"_index"`
	Type  string `json:"_type"`
}

func parseRequest(bulkableRequests []elastic.BulkableRequest) (*mapRequests, error) {

	header := true
	payload := &Payload{}
	requests := make(map[string]string)

	for _, bulkableRequest := range bulkableRequests {
		vv, err := bulkableRequest.Source()
		if err != nil {
			return nil, err
		}
		for _, v := range vv {
			if header {
				err := json.Unmarshal([]byte(v), payload)
				if err != nil {
					// skip error and try to parse next line
					continue
				}
				requests[payload.ID] = ""
				header = false
				continue
			}
			requests[payload.ID] = v
			header = true
		}
	}

	return &mapRequests{requests}, nil
}

func (p *mapRequests) ById(id string) string {
	request, exists := p.requests[id]
	if !exists {
		return "request not found"
	}
	return request
}

// BulkService ...
type BulkService struct {
	actions         int
	bulkService     *elastic.BulkService
	checkStatusCode []int
	initialTimeout  time.Duration
	requests        map[string]*elastic.BulkIndexRequest
	timeout         time.Duration
}

// Bulk creates a service
func Bulk(client *Elasticsearch, timeout time.Duration, actions int) *BulkService {
	return &BulkService{
		actions:         actions,
		bulkService:     elastic.NewBulkService(client.Client),
		checkStatusCode: []int{429},
		initialTimeout:  100 * time.Millisecond,
		requests:        make(map[string]*elastic.BulkIndexRequest, actions),
		timeout:         timeout,
	}
}

// Add adds bulkable requests, i.e. BulkIndexRequest, BulkUpdateRequest,
// and/or BulkDeleteRequest.
func (e BulkService) Add(index, tzpe string, msg interface{}) {
	id := uuid.New().String()
	r := elastic.NewBulkIndexRequest().Index(index).Type(tzpe).Doc(msg).Id(id)

	// TODO: create a PR for return a Bulkable request
	// then we can move this before Do() func
	// save requests for resending them, in case of failure
	e.requests[id] = r

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
	if err != nil {
		return nil, 0, true, err
	}

	// retrieve all failed responses for resending them again
	e.ResendRequests(bulkResponse.Failed(), e.checkStatusCode...)

	// reset requests
	e.requests = make(map[string]*elastic.BulkIndexRequest, e.actions)

	return bulkResponse, bulkResponse.Took, bulkResponse.Errors, err

}

// ResendRequests helps dealing with bulk rejections
// https://www.elastic.co/guide/en/elasticsearch/guide/current/_monitoring_individual_nodes.html
func (e BulkService) ResendRequests(bulkResponse []*elastic.BulkResponseItem, statusCode ...int) {
	if bulkResponse == nil {
		return
	}

	resendRequest := make([]elastic.BulkableRequest, 0, len(bulkResponse))

	// extract the rejected actions from the bulk response,
	for _, item := range bulkResponse {
		for _, status := range statusCode {
			if item.Status == status {
				resendRequest = append(resendRequest, e.requests[item.Id])
			}
		}
	}
	// pause the import thread for 3–5 seconds.
	time.Sleep(3 * time.Second)
	if len(resendRequest) > 0 {
		// send a new bulk request with just the rejected actions.
		e.bulkService.Add(resendRequest...)
	}

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
