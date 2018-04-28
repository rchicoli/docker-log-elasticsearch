package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/docker"
)

var logLevels = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
}

// StartLoggingRequest format
type StartLoggingRequest struct {
	File string      `json:"file,omitempty"`
	Info logger.Info `json:"info,omitempty"`
}

// StopLoggingRequest format
type StopLoggingRequest struct {
	File string `json:"file,omitempty"`
}

// CapabilitiesResponse format
type CapabilitiesResponse struct {
	Cap logger.Capability `json:"capabilities,omitempty"`
	Err string            `json:"err,omitempty"`
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

	h.HandleFunc("/LogDriver.StartLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StartLoggingRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(fmt.Errorf("error: could not decode payload: %v", err), w)
			return
		}

		if req.Info.ContainerID == "" {
			respond(errors.New("error: could not find containerID in request payload"), w)
			return
		}

		err := d.StartLogging(req.File, req.Info)

		// reader returns EOF if NoBody is sent
		respond(err, w)
	})

	h.HandleFunc("/LogDriver.StopLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StopLoggingRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := d.StopLogging(req.File)

		// reader returns EOF if NoBody is sent
		respond(err, w)
	})

	h.HandleFunc("/LogDriver.Capabilities", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&CapabilitiesResponse{
			Cap: logger.Capability{ReadLogs: false},
		})
	})

	// TODO: implement read logs from elasticsearch
	// h.HandleFunc("/LogDriver.ReadLogs", func(w http.ResponseWriter, r *http.Request) {
	// 	var req ReadLogsRequest
	// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 		http.Error(w, err.Error(), http.StatusBadRequest)
	// 		return
	// 	}

	// 	stream, err := d.ReadLogs(req.Info, req.Config)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	defer stream.Close()

	// 	w.Header().Set("Content-Type", "application/x-json-stream")
	// 	wf := ioutils.NewWriteFlusher(w)
	// 	io.Copy(wf, stream)

	// })

	if err := h.ServeUnix(d.Name(), 0); err != nil {
		log.WithError(err).Error("error: serving unix")
	}
}

type response struct {
	Err string `json:"err,omitempty"`
}

func respond(err error, w http.ResponseWriter) {
	var res response
	if err != nil {
		res.Err = err.Error()
	}
	json.NewEncoder(w).Encode(&res)
}
