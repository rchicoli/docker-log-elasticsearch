package elasticsearch

import (
	"golang.org/x/net/context"
)

type Client interface {
	Log(ctx context.Context, index, tzpe string, msg interface{}) error
}
