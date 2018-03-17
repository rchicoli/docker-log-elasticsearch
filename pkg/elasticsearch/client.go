package elasticsearch

import (
	"golang.org/x/net/context"
)

// Client ...
type Client interface {
	Log(ctx context.Context, index, tzpe string, msg interface{}) error

	// Stop stops the background processes that the client is running,
	// i.e. sniffing the cluster periodically and running health checks
	// on the nodes.
	Stop()
}
