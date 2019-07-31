/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/app/routes"
	"github.impcloud.net/RSP-Inventory-Suite/goplumber"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
)

func main() {
	mConfigurationError := metrics.GetOrRegisterGauge("DataProvider.Main.ConfigurationError", nil)
	mPipelineErr := metrics.GetOrRegisterGauge("DataProvider.Main.PipelineSetupError", nil)

	// Load config variables
	err := config.InitConfig()
	exitIfError(err, mConfigurationError, "Unable to load configuration variables.")

	setLogLevel()
	healthCheck(config.AppConfig.Port)
	// initMetrics()

	logMain := func(args ...interface{}) {
		log.WithFields(log.Fields{
			"Method": "main",
			"Action": "Start",
		}).Info(args...)
	}

	logMain("Starting Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exitIfError(loadPipelines(ctx), mPipelineErr, "Failed to start pipelines.")

	router := routes.NewRouter()
	startWebServer(router)

	log.WithField("Method", "main").Info("Completed.")
}

type uuidGen struct{}

func (uuidGen) Execute(ctx context.Context, w io.Writer, links map[string][]byte) error {
	_, err := w.Write([]byte(uuid.New()))
	return err
}

func loadPipelines(ctx context.Context) error {
	log.Debug("Starting pipelines.")
	plumber := goplumber.NewPlumber()

	// load pipelines and templates from the filesystem
	loader := goplumber.NewFSLoader(config.AppConfig.TemplatesDir)
	plumber.SetTemplateSource("template", loader)

	// add a task for getting Docker secrets
	plumber.SetSource("secret",
		goplumber.NewFSLoader(config.AppConfig.SecretsPath))

	// just use memory for K/V data; later, use consul or a db
	kvData := goplumber.NewMemoryStore()
	plumber.SetSource("get", kvData)
	plumber.SetSink("put", kvData)

	// add uuid generator
	plumber.SetClient("uuid", goplumber.PipeFunc(
		func(task *goplumber.Task) (goplumber.Pipe, error) { return uuidGen{}, nil }))

	log.Debug("Loading MQTT clients (if any).")
	pipedata := goplumber.NewFSLoader(config.AppConfig.PipelinesDir)
	for _, fn := range config.AppConfig.MQTTClients {
		name := fn
		if !strings.HasSuffix(fn, ".json") {
			fn += ".json"
		}
		mqttConfData, err := pipedata.GetFile(fn)
		if err != nil {
			return errors.Errorf("unable to load mqtt config %s", fn)
		}
		mc := &goplumber.MQTTClient{}
		if err := json.Unmarshal(mqttConfData, mc); err != nil {
			return errors.Wrapf(err, "unable to unmarshal mqtt config for %s", fn)
		}
		plumber.SetSink(name, mc)
	}

	log.Debug("Loading custom task types from pipelines.")
	for _, name := range config.AppConfig.CustomTaskTypes {
		data, err := pipedata.GetFile(name)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline from file %s", name)
		}

		var pipelineConf goplumber.PipelineConfig
		if err := json.Unmarshal(data, &pipelineConf); err != nil {
			return errors.Wrapf(err, "failed to unmarshal pipeline config from %s", name)
		}

		taskType, err := plumber.NewPipeline(&pipelineConf)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline %s", name)
		}
		client, err := goplumber.NewTaskType(taskType)
		if err != nil {
			return errors.Wrapf(err, "failed to create client for %s", name)
		}
		plumber.SetClient(pipelineConf.Name, client)
	}

	// only load the configured names
	log.Debug("Loading pipelines.")
	plines := map[*goplumber.Pipeline]time.Duration{}
	for _, name := range config.AppConfig.PipelineNames {
		data, err := pipedata.GetFile(name)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline %s", name)
		}

		var pipelineConf goplumber.PipelineConfig
		if err := json.Unmarshal(data, &pipelineConf); err != nil {
			return errors.Wrapf(err, "failed to unmarshal pipeline config from %s", name)
		}

		p, err := plumber.NewPipeline(&pipelineConf)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline %s", name)
		}

		d := pipelineConf.Trigger.Interval.Duration()
		if d <= 0 {
			old := d
			d = time.Duration(2) * time.Minute
			log.Warningf("setting pipeline '%s' interval from %s to %s",
				pipelineConf.Name, old, d)
		}
		plines[p] = d
	}

	log.Debugf("Starting %d pipelines.", len(plines))
	for p, d := range plines {
		go goplumber.RunPipelineForever(ctx, p, d)
	}
	return nil
}

func startWebServer(router http.Handler) {
	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           ":8080",
		Handler:        router,
		ReadTimeout:    900 * time.Second,
		WriteTimeout:   900 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// We want to report the listener is closed.
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the listener.
	go func() {
		log.Infof("%s running!", config.AppConfig.ServiceName)
		log.Infof("Listener closed: %v", server.ListenAndServe())
		wg.Done()
	}()

	// Listen for an interrupt signal from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt)

	// Wait for a signal to shutdown.
	<-osSignals

	// Create a context to attempt a graceful 5 second shutdown.
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {
		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "shutdown",
			"Timeout": timeout,
			"Message": err.Error(),
		}).Error("Graceful shutdown did not complete")

		// Looks like we timed out on the graceful shutdown. Kill it hard.
		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method":  "main",
				"Action":  "shutdown",
				"Message": err.Error(),
			}).Error("Error killing server")
		}
	}

	// Wait for the listener to report it is closed.
	wg.Wait()
}

func setLogLevel() {
	if config.AppConfig.LoggingLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10, // cfg.ReportingInterval,
			config.AppConfig.TelemetryEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}
