package elasticsearch

import (
	"context"
	"fmt"
	"time"

	elasticv1 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v1"
	elasticv2 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v2"
	elasticv5 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v5"
	elasticv6 "github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch/v6"
)

// Client ...
type Client interface {

	// NewBulkProcessorService(ctx context.Context, workers, bulkActions, bulkSize int, flushInterval time.Duration, stats bool) error
	// Stop the bulk processor and do some cleanup
	// Close() error
	// Flush() error

	// Stop stops the background processes that the client is running,
	// i.e. sniffing the cluster periodically and running health checks
	// on the nodes.
	Stop()

	Version() int
}

// Bulk Service implementation
type Bulk interface {
	Add(index, tzpe string, msg interface{})
	CommitRequired(actions int, bulkSize int) bool
	Do(ctx context.Context) (interface{}, int, bool, error)
	Errors(bulkResponse interface{}) []map[int]string
	EstimatedSizeInBytes() int64
	NumberOfActions() int
}

// NewClient ...
func NewClient(version string, url, username, password string, timeout time.Duration, sniff bool, insecure bool) (Client, error) {
	switch version {
	case "1":
		client, err := elasticv1.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "2":
		client, err := elasticv2.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "5":
		client, err := elasticv5.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	case "6":
		client, err := elasticv6.NewClient(url, username, password, timeout, sniff, insecure)
		if err != nil {
			return nil, fmt.Errorf("error: cannot create an elasticsearch client: %v", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("error: elasticsearch version not supported: %v", version)
	}
}

// NewBulk returns a bulkService depending on the client version
func NewBulk(client Client) (Bulk, error) {
	version := client.Version()
	switch version {
	case 1:
		return elasticv1.Bulk(client.(*elasticv1.Elasticsearch)), nil
	case 2:
		return elasticv2.Bulk(client.(*elasticv2.Elasticsearch)), nil
	case 5:
		return elasticv5.Bulk(client.(*elasticv5.Elasticsearch)), nil
	case 6:
		return elasticv6.Bulk(client.(*elasticv6.Elasticsearch)), nil
	default:
		return nil, fmt.Errorf("error: elasticsearch version not supported: %v", version)
	}
}
