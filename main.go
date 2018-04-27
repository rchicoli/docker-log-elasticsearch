package main

import (
	"fmt"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/docker"
)

var logLevels = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
}

type Handler struct {
	mux *http.ServeMux
}

func main() {

	levelVal := os.Getenv("LOG_LEVEL")
	if levelVal == "" {
		levelVal = "info"
	}
	if level, exists := logLevels[levelVal]; exists {
		log.SetLevel(level)
		log.SetFormatter(&log.TextFormatter{
			DisableTimestamp: true,
		})
	} else {
		fmt.Fprintln(os.Stderr, "invalid log level: ", levelVal)
		os.Exit(1)
	}

	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	d := docker.NewDriver()

	h.HandleFunc("/LogDriver.StartLogging", d.StartLogging)
	h.HandleFunc("/LogDriver.StopLogging", d.StopLogging)
	h.HandleFunc("/LogDriver.Capabilities", d.Capabilities)
	// h.HandleFunc("/LogDriver.ReadLogs", d.ReadLogs)

	if err := h.ServeUnix(d.Name(), 0); err != nil {
		log.WithError(err).Error("closing")
	}
}
