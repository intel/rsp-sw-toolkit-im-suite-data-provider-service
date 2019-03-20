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

package state

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/broker"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/config"
)

const defaultEndpointTmpl = `{{.Config.File.Endpoint}}`

var poller *Poller

// Poller wraps up all the dependencies, making testing easier.
//
// It's exposed as a state package variable, so all the configs are sharing this
// instance. Its values are generated at runtime from the application configuration.
// It's entrypoint for use is the PollAsync function, which sets the package
// variable from the service config and begins polling the server.
type Poller struct {
	baseEndpoint          string
	tmplValues            *serviceTmplVals
	mqttTopics            map[string]MqttTopicMapping
	dispatcherItemChannel *broker.ProviderItemChannel
	dispatcherErrChannel  *broker.ErrorChannel
	ccRequest             *CCRequest
}

// A ForcePollFunc tells the poller to immediately poll the server, rather than
// waiting until the next polling interval.
type ForcePollFunc func()

// NewPoller creates a new poller instance from the application's configuration.
// The package's poller variable should be set to that value for other portions
// of the application to correctly access it.
func NewPoller(conf config.ServiceConfig) (*Poller, error) {
	poller = &Poller{
		baseEndpoint: config.AppConfig.BaseEndpoint,
		tmplValues:   &serviceTmplVals{currentTimestamp: getCurrentTimestamp()},
		mqttTopics:   make(map[string]MqttTopicMapping),
	}

	for k, v := range conf.MqttTopicMap {
		poller.mqttTopics[k] = parseMqttTopicMapping(v)
	}

	poller.ccRequest = NewCCRequest(30, config.AppConfig.FailureRetries)

	// setup polling through CC, if configured
	if conf.ProxyThroughCloudConnector {
		poller.ccRequest.ProxyThroughCloudConnector(
			conf.CloudConnectorEndpoint,
			conf.OAuthEndpoint,
			conf.OAuthCredentials,
		)
	}

	return poller, nil
}

// SetDispatcherChannels saves the reference to the broker's channels
func (p *Poller) SetDispatcherChannels(errChan *broker.ErrorChannel, itemChan *broker.ProviderItemChannel) {
	p.dispatcherErrChannel = errChan
	p.dispatcherItemChannel = itemChan
}

// PollAsync asynchronously polls the configuration server and applies updates.
// Shut it down via the context.
func (p *Poller) PollAsync(ctx context.Context, config config.ServiceConfig) (ForcePollFunc, error) {

	pollForced := make(chan int, 1)
	forcePoll := func() {
		select {
		case pollForced <- 1:
		default:
		}
	}

	go func() {
		logrus.Debug("Starting desired state handler")

		failurePeriod := time.Second * time.Duration(config.FailureRetryPeriodSecs)
		pollPeriod := time.Second * time.Duration(config.PollPeriodSecs)
		var transientFails uint

		// do the initial poll
		result, _ := p.pollOnce()

		// until shutdown, wait a bit (based on the result), and poll again.
		for {
			var waitTime time.Duration
			switch result {
			case success, postApplyFailed, noNewManifest:
				waitTime = pollPeriod
				transientFails = 0
			case permanentFailure:
				// permanent failures need a new manifest to resolve, so we can
				// wait a (probably shorter) time and see if we get a new manifest
				waitTime = failurePeriod
				transientFails = 0
			case transientFailure:
				// transient failures can resolve on their own, but may take time.
				// we'll wait an increasing amount of time (up to 1 hr between requests).
				backoff := math.Min(float64(10*(uint(1)<<transientFails)), 3600)
				waitTime = time.Second * time.Duration(backoff)
				transientFails++
			}

			logrus.Debugf("Waiting %d seconds before next poll.", waitTime/time.Second)

			select {
			case <-ctx.Done():
				logrus.Debugf("Received done")
			case <-pollForced:
				logrus.Info("Polling early due to user request.")
				result, _ = p.pollOnce()
			case <-time.After(waitTime):
				result, _ = p.pollOnce()
			}
		}
	}()

	return forcePoll, nil
}

// serviceTmplVals are used to fill the URL templates.
type serviceTmplVals struct {
	lastQuery        int64
	currentTimestamp int64
}

// getCurrentTimestamp returns the current date/time in UNIX millisecond format
func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// pollOnce polls the server one time and process the resulting content.
func (p *Poller) pollOnce() (processResult, []error) {
	logrus.Debugf("Starting pollOnce")
	var result processResult
	var errors []error
	currTimestamp := getCurrentTimestamp()
	var wg sync.WaitGroup
	wg.Add(len(p.mqttTopics))

	//Query the endpoint defined for each of the mqttTopics
	for _, topic := range p.mqttTopics {
		go poll(p, topic, &wg, &errors)
	}

	wg.Wait()

	// determine the result
	if len(errors) > 0 {
		for _, err := range errors {
			// unwrap pollerErrors
			if pErr, ok := err.(pollerError); ok {
				result = pErr.result
			} else if result != permanentFailure {
				// assume a transientFailure if not otherwise marked
				result = transientFailure
			}
		}
	} else {
		result = success
	}

	// on success, move the in-process file to the success location
	if result == success {
		metrics.GetOrRegisterGauge("RFIDConfig.Polling.Successes", nil).Update(1)
		p.tmplValues.lastQuery = currTimestamp
	} else if result == permanentFailure {
		metrics.GetOrRegisterGauge("RFIDConfig.Polling.Failures", nil).Update(1)
	}
	logrus.Debugf("%s", errors)

	return result, errors
}

func poll(p *Poller, topicMap MqttTopicMapping, wg *sync.WaitGroup, errors *[]error) {
	defer wg.Done()
	metrics.GetOrRegisterGauge("RFIDConfig.Polling.Attempts", nil).Update(1)

	// execute the base template
	logrus.Debug("Filling base URL template.")
	baseURL, err := p.fillBaseURL(topicMap, p.tmplValues)
	if isError(err) {
		*errors = append(*errors, err)
	} else {
		logPollInfo("Polling server")
		var content []byte
		var err error
		content, err = p.ccRequest.PerformRequest(baseURL)

		logrus.Debugf("Content: %s", content)
		if isError(err) {
			logrus.Debugf("Error after polling")
			*errors = append(*errors, err)
		} else {
			topicList := topicMap.Topics
			for _, topic := range topicList {
				*p.dispatcherItemChannel <- &broker.ItemData{
					Type:  topic,
					Value: content,
				}
			}
			logPollInfo("Processing complete")
		}
	}
}

// fillBaseURL fills the URL and replaces parameters to query an endpoint
func (p *Poller) fillBaseURL(topic MqttTopicMapping, tmplValues *serviceTmplVals) (string, error) {
	// fill the baseURL, wrapping the tmplValues in a Service key for consistency
	baseURL := config.AppConfig.BaseEndpoint + topic.UrlTemplate

	baseURL = strings.Replace(baseURL, "{{lastQuery}}", strconv.FormatInt(tmplValues.lastQuery, 10), 1)
	baseURL = strings.Replace(baseURL, "{{currentTimestamp}}", strconv.FormatInt(tmplValues.currentTimestamp, 10), 1)

	return baseURL, nil
}

// isError checks and logs error return values. It returns true if err != nil.
func isError(err error, args ...interface{}) bool {
	if err == nil {
		return false
	}
	metrics.GetOrRegisterGauge("RFIDConfig.Polling.Errors", nil).Update(1)
	logrus.WithFields(logrus.Fields{
		"Method": "pollOnce",
		"Error":  fmt.Sprintf("%+v", err),
	}).Error(args...)
	return true
}

func logPollInfo(args ...interface{}) bool {
	logrus.WithFields(logrus.Fields{
		"Method": "handleDesiredState",
	}).Info(args...)
	return true
}

// processResult represents the result of polling for and processing manifest content.
type processResult int

const (
	noNewManifest    = processResult(iota) // there's no new manifest available
	permanentFailure                       // this manifest will never succeed
	transientFailure                       // this manifest might succeed at a later time
	postApplyFailed                        // the manifest application succeeded, but later steps failed
	success                                // the manifest was processed successfully
)

// pollerError is an error wrapper that includes a processResult.
type pollerError struct {
	error
	result processResult
}

// permanent marks an error as a permanent problem, i.e., one that is not expected
// to be resolved without manual involvement. For example, a manifest that fails
// schema validation is a permanent error, as it won't resolve on its own. If
// the error argument is already marked transient, it's changed to permanent.
func permanent(err error) error {
	if pErr, ok := err.(pollerError); ok {
		return pollerError{
			error:  pErr.error,
			result: permanentFailure,
		}
	}
	return pollerError{
		error:  err,
		result: permanentFailure,
	}
}

// transient marks an error as a transient problem, i.e., one that may resolve
// on its own, without user involvement. If the error argument is already marked
// permanent, it will not be changed to transient.
func transient(err error) error {
	if pErr, ok := err.(pollerError); ok {
		// don't change this to a transient failure if it was already marked permanent
		if pErr.result == permanentFailure {
			return pErr
		}
		return pollerError{
			error:  pErr.error,
			result: transientFailure,
		}
	}
	return pollerError{
		error:  err,
		result: transientFailure,
	}
}
