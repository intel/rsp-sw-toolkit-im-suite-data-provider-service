/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
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

package consulApi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type MockConsul struct {
	keyValueStore map[string]KeyValuePair
}

func NewMockConsul() *MockConsul {
	mock := MockConsul{}
	mock.keyValueStore = make(map[string]KeyValuePair)
	return &mock
}

var keyChannels map[string]chan bool

func (mock *MockConsul) Start() *httptest.Server {
	keyChannels = make(map[string]chan bool)

	testMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/v1/kv/") {
			key := strings.Replace(request.URL.Path, "/v1/kv/", "", 1)

			switch request.Method {
			case "PUT":
				body := make([]byte, request.ContentLength)
				if _, err := io.ReadFull(request.Body, body); err != nil {
					log.Printf("error reading request body: %s", err.Error())
				}

				keyValuePair, found := mock.keyValueStore[key]
				if found {
					keyValuePair.ModifyIndex++
					keyValuePair.Value = body
				} else {
					keyValuePair = KeyValuePair{
						Key:         key,
						Value:       body,
						ModifyIndex: 1,
						CreateIndex: 1,
						Flags:       0,
						LockIndex:   0,
					}
				}

				mock.keyValueStore[key] = keyValuePair

				log.Printf("PUTing new value for %s", key)
				channel, found := keyChannels[key]
				if found {
					channel <- true
				}

			case "GET":
				// this is what the wait query parameters will look like "index=1&wait=600000ms"
				query := request.URL.Query()
				waitTime := query.Get("wait")
				if waitTime != "" {
					waitForNextPut(key, waitTime)
				}
				keyValuePair, found := mock.keyValueStore[key]
				pairs := KeyValuePairs{&keyValuePair}
				if !found {
					http.NotFound(writer, request)
					return
				} else {
					jsonData, _ := json.MarshalIndent(&pairs, "", "  ")

					writer.Header().Set("Content-Type", "application/json")
					writer.WriteHeader(http.StatusOK)
					if _, err := writer.Write(jsonData); err != nil {
						log.Printf("error writing data response: %s", err.Error())
					}
				}
			}
		}
	}))

	return testMockServer
}

func waitForNextPut(key string, waitTime string) {
	timeout, err := time.ParseDuration(waitTime)
	if err != nil {
		log.Printf("Error parsing waitTime %s into a duration: %s", waitTime, err.Error())
	}
	channel := make(chan bool)
	keyChannels[key] = channel
	timedOut := false
	go func() {
		time.Sleep(timeout)
		timedOut = true
		if keyChannels[key] != nil {
			keyChannels[key] <- true
			log.Printf("Timed out watching for change on %s", key)
		}
	}()

	log.Printf("Watching for change on %s", key)
	<-channel
	close(channel)
	keyChannels[key] = nil
	if !timedOut {
		log.Printf("%s changed", key)
	}
}
