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
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/expect"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-mqtt-provider-service/app/config"
)

func init() {
	if testing.Verbose() {
		logrus.SetLevel(logrus.DebugLevel)
	}
	config.InitConfig()
}

func TestGenerateCurrentTimestamp(t *testing.T) {
	w := expect.WrapT(t)
	var testTimestamp int64
	w.ShouldBeSameType(testTimestamp, getCurrentTimestamp())
}

func TestFillBaseURL(t *testing.T) {
	w := expect.WrapT(t)
	testTopic := &MqttTopicMapping{
		UrlTemplate: "api/getInventory?from={{lastQuery}}&to={{currentTimestamp}}",
	}

	poller = &Poller{
		//urlGenerators: w.ShouldHaveResult(manifestio.NewURLGenGroup(defaultEndpointTmpl, base)).(manifestio.URLGenGroup),
		tmplValues: &serviceTmplVals{
			lastQuery:        1234,
			currentTimestamp: 5678,
		},
	}

	// filling the base should insert the ID
	baseURL := w.ShouldHaveResult(poller.fillBaseURL(*testTopic, poller.tmplValues)).(string)
	w.ShouldBeEqual(baseURL, "api/getInventory?from=1234&to=5678")
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
	poller = createTestPoller(t)
	result, errors := poller.pollOnce()
	w.ShouldBeEqual(result, errors)
}

// withPoller runs a functions within the context of a configured poller, which
// it cleans up before returning.
func withPoller(t *testing.T, f func()) {
	createTestPoller(t)

	f()
}

const (
	username = "bkrzanich"
	//nolint:gosec
	password = "cyclous_virtues"
)

func TestCreateTestPoller(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	poller := createTestPoller(t)
	w.ShouldBeEqual(poller.baseEndpoint, "https://api.myjson.com/bins")
}

func createTestPoller(t *testing.T) *Poller {
	w := expect.WrapT(t).StopOnMismatch()

	poller, err := NewPoller(config.AppConfig)
	w.ShouldBeNil(err)
	poller.ccRequest.SetClient(&http.Client{})
	return poller
}

func TestMqttTopicParsing(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	poller = &Poller{
		tmplValues: &serviceTmplVals{
			lastQuery:        1234,
			currentTimestamp: 5678,
		},
	}

	testMapping := getSampleMqttMapping()

	for _, topicMap := range testMapping {
		tm := parseMqttTopicMapping(topicMap)
		w.ShouldBeEqual(tm.UrlTemplate, "api/getInventory?from={{lastQuery}}&to={{currentTimestamp}}")
		w.ShouldBeEqual(len(tm.Topics), 2)
	}

}

func getSampleMqttMapping() map[string]interface{} {
	mqttStr := `{
		"inventory": {
			"urlTemplate": "api/getInventory?from={{lastQuery}}&to={{currentTimestamp}}",
			"topics": ["inventory", "productmasterdata"]
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
			"topics": ["productmasterdata"]
		  }
	  }`
	var testMapping map[string]interface{}
	json.Unmarshal([]byte(mqttStr), &testMapping)
	return testMapping
}
