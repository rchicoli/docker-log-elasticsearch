package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron"

	"golang.org/x/sync/errgroup"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"

	"github.com/rchicoli/docker-log-elasticsearch/pkg/elasticsearch"
	"github.com/rchicoli/docker-log-elasticsearch/pkg/regex"
)

const (
	name = "elasticsearchlog"
)

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

// ReadLogsRequest format
// type ReadLogsRequest struct {
// 	Info   logger.Info       `json:"info"`
// 	Config logger.ReadConfig `json:"config"`
// }

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

// Driver ...
type Driver struct {
	mu   *sync.Mutex
	ctx  context.Context
	logs map[string]*container
}

// NewDriver returns a pointer to driver
func NewDriver() *Driver {
	return &Driver{
		logs: make(map[string]*container),
		mu:   new(sync.Mutex),
	}
}

// Name return the docker plugin name
func (d *Driver) Name() string {
	return name
}

// StartLogging handler
func (d *Driver) StartLogging(w http.ResponseWriter, r *http.Request) {

	var req StartLoggingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(fmt.Errorf("error: could not decode payload: %v", err), w)
		return
	}

	if req.Info.ContainerID == "" {
		respond(errors.New("error: could not find containerID in request payload"), w)
		return
	}

	ctx := context.Background()

	c, err := d.newContainer(ctx, req.File)
	if err != nil {
		respond(err, w)
		return
	}
	c.info = req.Info
	c.logger = log.WithField("containerID", c.info.ContainerID)

	config := newConfiguration()
	if err := config.validateLogOpt(c.info.Config); err != nil {
		respond(err, w)
		return
	}

	c.esClient, err = elasticsearch.NewClient(config.version, config.url, config.username, config.password, config.timeout, config.sniff, config.insecure)
	if err != nil {
		respond(fmt.Errorf("error: cannot create an elasticsearch client: %v", err), w)
	}

	// org.elasticsearch.indices.InvalidIndexNameException: ... must be lowercase
	c.indexName = strings.ToLower(regex.ParseDate(time.Now(), config.index))
	if regex.IsValid(config.index) {
		c.cron = cron.New()
		c.cron.AddFunc("@daily", func() {
			d.mu.Lock()
			c.indexName = strings.ToLower(regex.ParseDate(time.Now(), config.index))
			d.mu.Unlock()
		})
		c.cron.Start()
	}

	var pctx context.Context
	c.pipeline.group, pctx = errgroup.WithContext(ctx)
	c.pipeline.inputCh = make(chan logdriver.LogEntry)
	c.pipeline.outputCh = make(chan LogMessage)

	if err := d.Read(pctx, req.File); err != nil {
		c.logger.WithError(err).Error("could not read line message")
	}

	if err := d.Parse(pctx, req.File, config.fields, config.grokMatch, config.grokPattern, config.grokPatternFrom, config.grokPatternSplitter, config.grokNamedCapture); err != nil {
		c.logger.WithError(err).Error("could not parse line message")
	}

	if err := d.Log(pctx, req.File, config.Bulk.workers, config.Bulk.actions, config.Bulk.size, config.Bulk.flushInterval, config.Bulk.stats, c.indexName, config.tzpe); err != nil {
		c.logger.WithError(err).Error("could not log to elasticsearch")
	}

	respond(err, w)

}

// StopLogging handler
func (d *Driver) StopLogging(w http.ResponseWriter, r *http.Request) {

	var req StopLoggingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// this is required for some environment like travis
	// otherwise the start and stop function are executed
	// too fast, even before messages are sent to the pipeline
	time.Sleep(1 * time.Second)

	c, err := d.getContainer(req.File)
	if err != nil {
		respond(err, w)
		return
	}

	filename := path.Base(req.File)
	d.mu.Lock()
	delete(d.logs, filename)
	d.mu.Unlock()

	c.logger.WithField("fifo", req.File).Debug("removing fifo file")

	if c.stream != nil {
		c.logger.Info("closing container stream")
		c.stream.Close()
	}

	if c.pipeline.group != nil {
		c.logger.Info("closing pipeline")

		// Check whether any goroutines failed.
		err := c.pipeline.group.Wait()
		if err != nil {
			c.logger.WithError(err).Error("pipeline wait group")
		}
	}

	if c.cron != nil {
		c.cron.Stop()
	}

	// if c.esClient != nil {
	//	close client connection on last pipeline
	// }

	respond(err, w)

}

// Capabilities handler
func (d *Driver) Capabilities(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(&CapabilitiesResponse{
		Cap: logger.Capability{ReadLogs: false},
	})
}

// func (d *Driver) ReadLogs(w http.ResponseWriter, r *http.Request) {
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

// }
