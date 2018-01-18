package main

import (
	"fmt"
	"os"

	"github.com/docker/go-plugins-helpers/sdk"
	"github.com/sirupsen/logrus"

	"github.com/rchicoli/docker-log-elasticsearch/api"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/docker"
)

var logLevels = map[string]logrus.Level{
	"debug": logrus.DebugLevel,
	"info":  logrus.InfoLevel,
	"warn":  logrus.WarnLevel,
	"error": logrus.ErrorLevel,
}

func main() {
	levelVal := os.Getenv("LOG_LEVEL")
	if levelVal == "" {
		levelVal = "info"
	}
	if level, exists := logLevels[levelVal]; exists {
		logrus.SetLevel(level)
	} else {
		fmt.Fprintln(os.Stderr, "invalid log level: ", levelVal)
		os.Exit(1)
	}

	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	d := docker.NewDriver()
	api.Handlers(&h, d)
	if err := h.ServeUnix(d.Name(), 0); err != nil {
		panic(err)
	}
}
