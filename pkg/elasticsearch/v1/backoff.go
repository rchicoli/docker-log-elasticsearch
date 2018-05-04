package v1

import (
	"errors"
	"net/http"
	"syscall"
	"time"

	"gopkg.in/olivere/elastic.v2"
)

// MyRetrier ,,,
type MyRetrier struct {
	backoff elastic.Backoff
}

// NewMyRetrier ...
func NewMyRetrier(timeout time.Duration) *MyRetrier {
	return &MyRetrier{
		backoff: elastic.NewExponentialBackoff(100*time.Millisecond, timeout),
	}
}

// Retry ...
func (r *MyRetrier) Retry(retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	// Fail hard on a specific error
	if err == syscall.ECONNREFUSED {
		return 0, false, errors.New("network problems: connection refused")
	}

	// Let the backoff strategy decide how long to wait and whether to stop
	wait, stop := r.backoff.Next(retry)

	return wait, stop, nil
}
