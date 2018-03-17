package main

import (
	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/docker"
)

func main() {

	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	d := docker.NewDriver()
	docker.Handlers(&h, d)
	if err := h.ServeUnix(d.Name(), 0); err != nil {
		panic(err)
	}
}
