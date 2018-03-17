package docker

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/docker/docker/daemon/logger"
	"github.com/docker/go-plugins-helpers/sdk"
)

// StartLoggingRequest ...
type StartLoggingRequest struct {
	File string
	Info logger.Info
}

// StopLoggingRequest ...
type StopLoggingRequest struct {
	File string
}

// CapabilitiesResponse ...
type CapabilitiesResponse struct {
	Err string
	Cap logger.Capability
}

// ReadLogsRequest ...
type ReadLogsRequest struct {
	Info   logger.Info
	Config logger.ReadConfig
}

// LogDriver ...
type LogDriver interface {
	StartLogging(string, logger.Info) error
	StopLogging(string) error
}

// Handlers ...
func Handlers(h *sdk.Handler, d LogDriver) {
	h.HandleFunc("/LogDriver.StartLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StartLoggingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Info.ContainerID == "" {
			respond(errors.New("must provide container id in log context"), w)
			return
		}

		err := d.StartLogging(req.File, req.Info)
		respond(err, w)
	})

	h.HandleFunc("/LogDriver.StopLogging", func(w http.ResponseWriter, r *http.Request) {
		var req StopLoggingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := d.StopLogging(req.File)
		respond(err, w)
	})

	h.HandleFunc("/LogDriver.Capabilities", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&CapabilitiesResponse{
			Cap: logger.Capability{ReadLogs: false},
		})
	})

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
}

type response struct {
	Err string
}

func respond(err error, w http.ResponseWriter) {
	var res response
	if err != nil {
		res.Err = err.Error()
	}
	json.NewEncoder(w).Encode(&res)
}
