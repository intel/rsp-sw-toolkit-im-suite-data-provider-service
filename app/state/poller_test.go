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
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/app/broker"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/expect"
)

func init() {
	if testing.Verbose() {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}
}

func TestFillBaseURL(t *testing.T) {
	w := expect.WrapT(t)
	testTopic := &MqttTopicMapping{
		UrlTemplate: "/api/getInventory?siteId={{siteID}}&from={{lastQuery}}&to={{currentTimestamp}}",
	}

	poller = &Poller{
		tmplValues: &serviceTmplVals{
			siteID:           "mySite",
			lastQuery:        time.Unix(1236545064, 543987911),
			currentTimestamp: time.Unix(5674509978, 128334715),
		},
	}

	// filling the base should insert the ID
	baseURL := w.ShouldHaveResult(poller.fillBaseURL(*testTopic, poller.tmplValues)).(string)
	w.ShouldBeEqual(baseURL, poller.baseEndpoint+"/api/getInventory?siteId=mySite&from=2009-03-08T20:44:24.543Z&to=2149-10-26T04:46:18.128Z")
	w.Log(baseURL)
}

func TestConfigProxyThroughCC(t *testing.T) {
	withPoller(t, func() {
		w := expect.WrapT(t)
		w.ShouldBeFalse(poller.ccRequest.IsProxied())
		poller.ccRequest.ProxyThroughCloudConnector(
			"http://cloudconn.com", "http://oauth.com", "username:password",
		)
		w.ShouldBeTrue(poller.ccRequest.IsProxied())
	})
}

func TestPoller_pollOnce(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	contentServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		w.Logf("content server called with: %s", r.URL)
		w.ShouldHaveResult(rw.Write([]byte(``)))
	}))
	defer contentServer.Close()

	var dispatcher localDispatcher
	dispatcher.internalErrChannel = make(broker.ErrorChannel, 1)
	dispatcher.internalItemChannel = make(broker.ProviderItemChannel, 1)
	conf := config.AppConfig
	conf.ProxyThroughCloudConnector = false
	conf.BaseEndpoint = contentServer.URL
	poller, _ := NewPoller(conf)
	poller.SetDispatcherChannels(&dispatcher.internalErrChannel, &dispatcher.internalItemChannel)
	poller.ccRequest.SetClient(contentServer.Client())

	result, errors := poller.pollOnce()
	if len(errors) != 0 {
		w.Log(errors)
	}
	w.ShouldBeEqual(result, success)
	w.ShouldBeEqual(len(errors), 0)
}

// withPoller runs a functions within the context of a configured poller, which
// it cleans up before returning.
func withPoller(t *testing.T, f func()) {
	createTestPoller(t, config.AppConfig)

	f()
}

const (
	username = "bkrzanich"
	//nolint:gosec
	password = "cyclous_virtues"
)

func TestCreateTestPoller(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	poller := createTestPoller(t, config.AppConfig)
	w.ShouldBeEqual(poller.baseEndpoint, "https://api.myjson.com/bins")
}

type localDispatcher struct {
	// Internal for the dispatcher goroutine
	internalErrChannel  broker.ErrorChannel
	internalItemChannel broker.ProviderItemChannel

	itemChannels map[string][]*broker.ItemChannel
	errChannels  map[string][]*broker.ErrorChannel
	// Synchronize add/remove access to the channel maps
	chanMutex sync.Mutex
}

func createTestPoller(t *testing.T, conf config.ServiceConfig) *Poller {
	w := expect.WrapT(t).StopOnMismatch()

	var dispatcher localDispatcher
	dispatcher.internalErrChannel = make(broker.ErrorChannel, 1)
	dispatcher.internalItemChannel = make(broker.ProviderItemChannel, 1)

	poller, err := NewPoller(conf)
	poller.SetDispatcherChannels(&dispatcher.internalErrChannel, &dispatcher.internalItemChannel)

	w.ShouldBeNil(err)
	poller.ccRequest.SetClient(&http.Client{})
	return poller
}

func TestMqttTopicParsing(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	poller = &Poller{
		tmplValues: &serviceTmplVals{
			lastQuery:        time.Unix(1236545064, 543987911),
			currentTimestamp: time.Unix(5674509978, 128334715),
		},
	}

	testMapping := getSampleMqttMapping()

	for _, topicMap := range testMapping {
		tm := parseMqttTopicMapping(topicMap)
		w.ShouldBeEqual(tm.UrlTemplate, "/api/getInventory?from={{lastQuery}}&to={{currentTimestamp}}")
		w.ShouldBeEqual(len(tm.Topics), 2)
	}

}

func getSampleMqttMapping() map[string]interface{} {
	mqttStr := `{
		"inventory": {
			"urlTemplate": "/api/getInventory?from={{lastQuery}}&to={{currentTimestamp}}",
			"topics": ["inventory", "productmasterdata"],
			"useAuth": true
		  }
	  }`
	var testMapping map[string]interface{}
	json.Unmarshal([]byte(mqttStr), &testMapping)
	return testMapping
}

func getSampleMqttMapping2() map[string]interface{} {
	mqttStr := `{
		  "skumapping": {
			"urlTemplate": "/1g81wy",
			"topics": ["productmasterdata"],
			"useAuth": true
		  }
	  }`
	var testMapping map[string]interface{}
	json.Unmarshal([]byte(mqttStr), &testMapping)
	return testMapping
}
