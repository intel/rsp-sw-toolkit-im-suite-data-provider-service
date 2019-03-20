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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/broker"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/state"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/pkg/middlewares"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/pkg/web"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	reporter "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics-influxdb"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/routes"
)

type localDispatcher struct {
	// Internal for the dispatcher goroutine
	internalErrChannel  broker.ErrorChannel
	internalItemChannel broker.ProviderItemChannel

	itemChannels map[string][]*broker.ItemChannel
	errChannels  map[string][]*broker.ErrorChannel
	// Synchronize add/remove access to the channel maps
	chanMutex sync.Mutex
}

func main() {
	mConfigurationError := metrics.GetOrRegisterGauge("RFIDConfig.Main.ConfigurationError", nil)
	mPollingError := metrics.GetOrRegisterGauge("RFIDConfig.Main.PollingSetupError", nil)

	// Load config variables
	err := config.InitConfig()
	exitIfError(err, mConfigurationError, "Unable to load configuration variables.")

	setLogLevel()
	healthCheck(config.AppConfig.Port)
	initMetrics()

	logMain := func(args ...interface{}) {
		log.WithFields(log.Fields{
			"Method": "main",
			"Action": "Start",
		}).Info(args...)
	}

	logMain("Starting RFID Config Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Debug("Starting poller.")
	poller, err := state.NewPoller(config.AppConfig)
	log.Debug("Starting RFID broker")
	initBroker(poller)
	pollNow, err := poller.PollAsync(ctx, config.AppConfig)
	exitIfError(err, mPollingError, "Failed to start polling.")

	// create a route for force-polling the server
	router := routes.NewRouter()
	pollNowHandler := web.Handler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		pollNow()
		web.Respond(ctx, w, "sending poll request", http.StatusOK)
		return nil
	})
	pollNowHandler = middlewares.Recover(middlewares.Logger(pollNowHandler))
	router.Methods("POST").Path("/pollnow").
		Name("PollNow").Handler(pollNowHandler)

	startWebServer(router)
	log.WithField("Method", "main").Info("Completed.")
}

func initBroker(pllr *state.Poller) {
	onBrokerStarted := make(broker.BrokerStartedChannel, 1)
	onBrokerError := make(broker.ErrorChannel, 1)

	providerOptions := broker.MosquittoProviderOptions{
		Gateway:                  config.AppConfig.Gateway,
		Topics:                   []string{"inventory"}, //Add list of topics to listen and publish to here
		AllowSelfSignedCerts:     config.AppConfig.SkipCertVerify,
		EncryptGatewayConnection: config.AppConfig.EncryptGatewayConnection,
		OnStarted:                onBrokerStarted,
		OnError:                  onBrokerError,
	}

	if config.AppConfig.GatewayCredentialsPath != "" {
		username, password, err := readCredentialsFile(config.AppConfig.GatewayCredentialsPath)
		if err != nil {
			log.Fatalln("Credentials file error:", err)
		}

		providerOptions.Username = username
		providerOptions.Password = password
	}

	go routeAndPublishMqtt(&providerOptions, pllr)
}

func routeAndPublishMqtt(options *broker.MosquittoProviderOptions, pllr *state.Poller) {
	rfidBroker := broker.NewMosquittoClient(options)

	var dispatcher localDispatcher
	dispatcher.internalErrChannel = make(broker.ErrorChannel, 10)
	dispatcher.internalItemChannel = make(broker.ProviderItemChannel, 10)

	pllr.SetDispatcherChannels(&dispatcher.internalErrChannel, &dispatcher.internalItemChannel)

	rfidBroker.Start(dispatcher.internalItemChannel, dispatcher.internalErrChannel)
	for {
		select {
		case started := <-options.OnStarted:
			if !started.Started {
				log.WithFields(log.Fields{
					"Method": "main",
					"Action": "connecting to mosquitto broker",
					"Host":   config.AppConfig.Gateway,
				}).Fatal("Mosquitto broker has failed to start")
			}

			log.Info("Mosquitto broker has started")

		case item, ok := <-dispatcher.internalItemChannel:
			if ok {
				rfidBroker.Publish(item.Type, item.Value.([]byte))
			} else {
				dispatcher.internalItemChannel = nil
			}

		case err := <-options.OnError:
			log.WithFields(log.Fields{
				"Method": "main",
				"Action": "Receiving sensing error exiting",
			}).Fatal(err)

		}

	}
}

// readCredentialsFile obtains the username/id and password from the specified path
// the file format is: "<username/id>\t<password>"
func readCredentialsFile(path string) (id, password string, err error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	contents := strings.SplitN(string(buf), "\t", 2)
	if len(contents) == 2 {
		id = strings.TrimSpace(contents[0])
		password = strings.TrimSpace(contents[1])
	} else {
		err = fmt.Errorf("invalid credentials file (%s) contents", path)
	}

	return
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
