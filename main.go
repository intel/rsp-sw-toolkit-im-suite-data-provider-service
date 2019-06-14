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
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/goplumber"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/app/routes"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	reporter "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics-influxdb"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/app/config"
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

func loadPipelines(ctx context.Context) error {
	log.Debug("Starting pipeline runner.")
	runner := goplumber.NewPipelineRunner()

	// load templates from the filesystem
	templateLoader := goplumber.NewFSLoader(config.AppConfig.TemplatesDir)
	plumber := goplumber.NewPlumber(templateLoader)

	// add a task for getting Docker secrets
	plumber.TaskGenerators["secret"] = goplumber.NewLoadTaskGenerator(
		goplumber.NewFSLoader(config.AppConfig.SecretsPath))

	// just use memory for K/V data; later, use consul
	kvData := goplumber.NewMemoryStore()
	plumber.TaskGenerators["get"] = goplumber.NewLoadTaskGenerator(kvData)
	plumber.TaskGenerators["put"] = goplumber.NewStoreTaskGenerator(kvData)

	// load pipelines from the filesystem
	pipedata := goplumber.NewFSLoader(config.AppConfig.PipelinesDir)

	// only load the configured names
	log.Debug("Loading pipelines.")
	for _, name := range config.AppConfig.PipelineNames {
		data, err := pipedata.GetFile(name)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline %s", name)
		}

		p, err := plumber.NewPipeline(data)
		if err != nil {
			return errors.Wrapf(err, "failed to load pipeline %s", name)
		}

		runner.AddPipeline(ctx, p)
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
